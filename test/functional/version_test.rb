require 'minitest/autorun'

require 'lib/cmd'

class TestVersion < Minitest::Test
  def test_simple_run
    cmd = GitWho.new
    cmd.run
    assert cmd.success?
  end
end
