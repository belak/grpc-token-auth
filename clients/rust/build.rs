fn main() {
    tonic_build::compile_protos("../../pb/echo.proto").unwrap();
}
