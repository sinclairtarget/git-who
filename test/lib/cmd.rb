require 'open3'
require 'pathname'

class GitWhoError < StandardError
end

class GitWho
  def initialize(exec_path, rundir)
    @exec_path = exec_path
    @rundir = rundir
  end

  def run(*args)
    stdout_s, stderr_s, status = Open3.capture3(
      @exec_path,
      *args,
      chdir: @rundir,
    )

    unless status.success?
      raise GitWhoError(
        "Command failed with status: #{status.exitstatus}\n#{stderr_s}"
      )
    end

    stdout_s
  end

  def self.built_bin_path
    p = Pathname.new(__dir__) + '../../git-who'
    p.cleanpath.to_s
  end
end
