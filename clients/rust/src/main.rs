#[macro_use]
extern crate log;

use anyhow::Context;
use futures::FutureExt;
use http::Uri;
use tokio::sync::mpsc;
use tonic::transport::{Channel, ClientTlsConfig};
use tonic::Request;

pub mod proto {
    tonic::include_proto!("echo");
}

use crate::proto::echo_service_client::EchoServiceClient;
use crate::proto::EchoRequest;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    pretty_env_logger::init();

    let token = std::env::var("TOKEN").context("missing TOKEN variable")?;
    let url = std::env::var("ECHO_URL").unwrap_or_else(|_| "http://localhost:8000".to_string());

    let uri: Uri = url.parse().context("failed to parse ECHO_URL")?;
    let mut channel_builder = Channel::builder(uri.clone());

    match uri.scheme_str() {
        None | Some("https") => {
            debug!("Enabling TLS");
            channel_builder =
                channel_builder.tls_config(ClientTlsConfig::new().domain_name(uri.host().unwrap()));
        }
        _ => {}
    }

    let channel = channel_builder
        .connect()
        .await
        .context("Failed to connect to echo service")?;

    let auth_header = format!("Bearer {}", token).parse()?;

    let mut client = EchoServiceClient::with_interceptor(channel, move |mut req: Request<()>| {
        req.metadata_mut()
            .insert("authorization", auth_header);
        Ok(req)
    });

    let resp = client
        .echo(EchoRequest {
            message: "hello world".to_string(),
        })
        .await?
        .into_inner();

    info!("got echo response: {}", resp.message);

    let (outbound_sender, outbound_receiver) = mpsc::channel(10);

    futures::future::try_join_all(vec![
        handle_inbound(client, outbound_receiver).boxed(),
        handle_outbound(outbound_sender).boxed(),
    ])
    .await?;

    Ok(())
}

async fn handle_inbound(
    mut client: EchoServiceClient<Channel>,
    outbound_receiver: mpsc::Receiver<crate::proto::EchoRequest>,
) -> anyhow::Result<()> {
    debug!("starting stream");

    let mut input = client
        .streaming_echo(Request::new(outbound_receiver))
        .await?
        .into_inner();

    debug!("got stream");

    while let Some(resp) = input.message().await? {
        info!("got streaming echo response: {}", resp.message);
    }

    Ok(())
}

async fn handle_outbound(
    mut output: mpsc::Sender<crate::proto::EchoRequest>,
) -> anyhow::Result<()> {
    for i in 0..5 {
        debug!("sending stream request: {}", i);

        output
            .send(EchoRequest {
                message: format!("echo request {}", i),
            })
            .await?;
    }

    Ok(())
}
