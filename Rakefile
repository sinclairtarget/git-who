require 'fileutils'
require 'rake/clean'

PROGNAME = 'git-who'
SUPPORTED = [
  ['darwin', 'arm64'],
  ['darwin', 'amd64'],
  ['linux', 'amd64'],
  ['linux', 'arm64'],
  ['linux', 'arm'],
  ['windows', 'amd64'],
]
OUTDIR = 'out'
RELEASE_DIRS = SUPPORTED.map do |os, arch|
    "#{OUTDIR}/#{os}_#{arch}"
end

task default: [:build]

desc 'Run go fmt'
task :fmt do
  sh 'go fmt ./internal/...'
  sh 'go fmt *.go'
end

desc 'Build executable'
task :build do
  gohostos = `go env GOHOSTOS`.strip
  gohostarch = `go env GOHOSTARCH`.strip
  build_for_platform gohostos, gohostarch, out: exec_name(gohostos)
end

namespace 'release' do
  directory OUTDIR

  RELEASE_DIRS.each do |dir|
    directory dir
  end

  desc 'Build binaries for all supported platforms'
  task build: RELEASE_DIRS do
    SUPPORTED.each do |os, arch|
      output_dir = "#{OUTDIR}/#{os}_#{arch}"
      progname = exec_name(os)
      build_for_platform(os, arch, out: "#{output_dir}/#{progname}")

      version = get_version
      sh "tar czf #{OUTDIR}/gitwho_#{version}_#{os}_#{arch}.tar.gz "\
        "-C #{OUTDIR} #{os}_#{arch}"
    end
  end

  desc 'Sign checksum of built artifacts'
  task :sign do
    FileUtils.cd(OUTDIR) do
      version = get_version
      sumsfile = "SHA2-256SUMS_#{version}.txt"
      sh "shasum -a 256 **/git-who* > #{sumsfile}"
      sh "ssh-keygen -Y sign -n file -f ~/.ssh/gitwho_ed25519 #{sumsfile}"
    end
  end

  task all: [:build, :sign]
end

CLOBBER.include(OUTDIR)
CLOBBER.include(PROGNAME)

def get_version()
  `git describe --tags --always --dirty`.strip
end

def get_commit()
  `git rev-parse --short HEAD`.strip
end

def exec_name(goos)
  if goos == 'windows'
    PROGNAME + '.exe'
  else
    PROGNAME
  end
end

def build_for_platform(goos, goarch, out: PROGNAME)
  version = get_version
  rev = get_commit
  sh "GOOS=#{goos} GOARCH=#{goarch} go build -a -o #{out} "\
    "-ldflags '-s -w -X main.Commit=#{rev} -X main.Version=#{version}'"
end

namespace 'test' do
  desc 'Run all tests'
  task all: [:unit, :integration, :functional]

  desc 'Run unit tests'
  task :unit do
    sh 'go test -count=1 ./internal/...'
  end

  desc 'Run integration tests'
  task :integration do
    sh 'go test -count=1 ./test/integration/...'
  end

  begin
    require 'minitest/test_task'

    Minitest::TestTask.create(:functional) do |t|
      t.libs << "test/functional"
      t.libs << "test/functional/lib"
      t.test_globs = ["test/functional/**/*_test.rb"]
    end
  rescue LoadError
    task :functional do
      puts "Skipping functional tests; minitest gem not found."
    end
  end
end

task test: 'test:all'
