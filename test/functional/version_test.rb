require 'minitest/autorun'

require 'lib/cmd'
require 'lib/repo'

class TestVersion < Minitest::Test
  def test_version
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run '--version'
    assert stdout_s
  end
end
