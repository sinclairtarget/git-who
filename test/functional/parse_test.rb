require 'minitest/autorun'

require 'lib/cmd'
require 'lib/repo'

# Test parse subcommand.
class TestTable < Minitest::Test
  def test_parse
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'parse'
    refute_empty(stdout_s)
  end

  def test_parse_shortlog
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'parse', '-s'
    refute_empty(stdout_s)
  end
end
