task default: [:build]

desc "Run all unit tests"
task :test do
  sh "go test ./internal/..."
end

desc "Run go fmt"
task :fmt do
  sh "go fmt ./internal/..."
  sh "go fmt *.go"
end

desc "Build executable"
task :build do
  rev = `git rev-parse --short HEAD`.strip
  version = `git describe --tags --exact-match --always --dirty`.strip
  sh "go build -a -ldflags '-s -w -X main.Commit=#{rev}"\
    " -X main.Version=#{version}'"
end

desc "Delete executable"
task :clean do
  sh "go clean"
end
