require 'minitest/autorun'

require 'lib/cmd'
require 'lib/repo'

# This set of tests for the `tree` subcommand does nothing to check the
# validity of the output. We just try to hit as many codepaths as we can to
# check that the program doesn't error out.
class TestTree < Minitest::Test
  SHOW_ALL_FLAGS = ['', '-a']
  MODE_FLAGS = ['', '-c', '-f', '-l', '-m']
  EMAIL_FLAGS = ['', '-e']
  MERGES_FLAGS = ['', '--merges']

  AUTHOR_FILTER_FLAGS = ['', '--author Bob']
  NAUTHOR_FILTER_FLAGS = ['', '--nauthor Alice']
  SINCE_FILTER_FLAGS = ['', '--since 2024-12-25']
  UNTIL_FILTER_FLAGS = ['', '--until 2025-02-01']

  def test_tree_no_flags
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree'
    refute_empty(stdout_s)
  end

  def test_tree_subdir
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree', 'file-rename'
    refute_empty(stdout_s)
  end

  def test_tree_exclude_ext_pathspec
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree', '--', 'exclude-ext', ':!*.py'
    refute_empty(stdout_s)
  end

  def test_tree_exclude_ext_pathspec_longform
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree', '--', 'exclude-ext', ':(exclude)*.py'
    refute_empty(stdout_s)
  end

  def test_tree_exclude_ext_pathspec_no_concurrent
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree', '--', 'exclude-ext', ':!*.py', n_procs: 1
    refute_empty(stdout_s)
  end

  def test_exclude_ext_pathspec_trailing_slash
    skip("git ls-files bug")

    # This doesn't work because ls-files doesn't output anything. But it works
    # if you put the exclude magic pathspec first. Bug with git ls-files?
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'tree', '--', 'exclude-ext/', ':!*.py'
    refute_empty(stdout_s)
  end

  all_flag_combos = GitWho.generate_args_cartesian_product([
    SHOW_ALL_FLAGS,
    MODE_FLAGS,
    EMAIL_FLAGS,
    MERGES_FLAGS,
  ])
  all_flag_combos.each do |flags|
    test_name = "test_tree_(#{flags.join ','})"
    define_method(test_name) do
      cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
      stdout_s = cmd.run 'tree', *flags
      assert stdout_s
    end
  end

  all_filter_flag_combos = GitWho.generate_args_cartesian_product([
    AUTHOR_FILTER_FLAGS,
    NAUTHOR_FILTER_FLAGS,
    SINCE_FILTER_FLAGS,
    UNTIL_FILTER_FLAGS,
  ])
  all_filter_flag_combos.each do |flags|
    test_name = "test_tree_filter_(#{flags.join ','})"
    define_method(test_name) do
      cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
      stdout_s = cmd.run 'tree', *flags
      refute_empty(stdout_s)
    end
  end
end
