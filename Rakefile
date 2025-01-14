require 'fileutils'
require 'rake/clean'

PROGNAME = 'git-who'
SUPPORTED = [
  ['darwin', 'arm64'],
  ['darwin', 'amd64'],
  ['linux', 'amd64'],
  ['linux', 'arm64'],
  ['linux', 'arm'],
]
OUTDIR = 'out'
RELEASE_DIRS = SUPPORTED.map do |os, arch|
    "#{OUTDIR}/#{os}_#{arch}"
end

task default: [:build]

desc 'Run all unit tests'
task :test do
  sh 'go test ./internal/...'
end

desc 'Run go fmt'
task :fmt do
  sh 'go fmt ./internal/...'
  sh 'go fmt *.go'
end

desc 'Build executable'
task :build do
  gohostos = `go env GOHOSTOS`.strip
  gohostarch = `go env GOHOSTARCH`.strip
  build_for_platform gohostos, gohostarch
end

namespace 'release' do
  directory OUTDIR

  RELEASE_DIRS.each do |dir|
    directory dir
  end

  desc 'Build binaries for all supported platforms'
  task build: RELEASE_DIRS do
    SUPPORTED.each do |os, arch|
      output_dir = "out/#{os}_#{arch}"
      build_for_platform(os, arch, out: "#{output_dir}/#{PROGNAME}")
    end
  end
end

CLOBBER.include(OUTDIR)
CLOBBER.include(PROGNAME)

def build_for_platform(goos, goarch, out: 'git-who')
  rev = `git rev-parse --short HEAD`.strip
  version = `git describe --tags --always --dirty`.strip
  sh "GOOS=#{goos} GOARCH=#{goarch} go build -a -o #{out} "\
    "-ldflags '-s -w -X main.Commit=#{rev} -X main.Version=#{version}'"
end
