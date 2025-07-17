require 'open3'
require 'pathname'

BIN_RELPATH = '../../../git-who'

class GitWhoError < StandardError
end

class GitWho
  def initialize(exec_path, rundir)
    @exec_path = exec_path
    @rundir = rundir
  end

  def run(*args, cache_home: nil, n_procs: nil)
    env_hash = {}

    if cache_home
      env_hash['XDG_CACHE_HOME'] = cache_home
    else
      env_hash['GIT_WHO_DISABLE_CACHE'] = '1'
    end

    unless n_procs.nil?
      env_hash['GOMAXPROCS'] = n_procs.to_s
    end

    split_args = args.reduce([]) do |args, arg|
      arg.split(' ').each do |part|
        args << part
      end

      args
    end

    stdout_s, stderr_s, status = Open3.capture3(
      env_hash,
      @exec_path,
      *split_args,
      chdir: @rundir,
    )

    unless status.success?
      invocation = GitWho.format_invocation(split_args)
      raise GitWhoError,
        "#{invocation} exited with status: #{status.exitstatus}\n#{stderr_s}"
    end

    stdout_s
  end

  def self.built_bin_path
    p = Pathname.new(__dir__) + BIN_RELPATH
    p.cleanpath.to_s
  end

  def self.format_invocation(args)
    'git-who ' + args.join(' ')
  end

  # Given a list of "flagsets", where each flagset is a set of mutually
  # exclusive flags that could be supplied for a command, returns the cartesian
  # product of all the flagsets (i.e. all possible combinations of flags).
  def self.generate_args_cartesian_product(flagsets, no_empty: true)
    all_args =
      if flagsets.empty?
        [[]]
      else
        head = flagsets[0]
        tail = flagsets[1..]

        tail_args = self.generate_args_cartesian_product(tail, no_empty: false)

        head.reduce([]) do |all_args, flag|
          tail_args.each do |args|
            if flag.empty?
              all_args << args
            else
              all_args << [flag] + args
            end
          end

          all_args
        end
      end

    if no_empty
      all_args.filter { |args| !args.empty? }
    else
      all_args
    end
  end
end
