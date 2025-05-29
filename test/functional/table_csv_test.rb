require 'minitest/autorun'
require 'csv'

require 'lib/cmd'
require 'lib/repo'

class TestTableCSV < Minitest::Test
  def test_table_csv
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'table', '--csv'
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    assert_equal data.headers, [
      'name', 'commits', 'last commit time', 'first commit time',
    ]
    assert_equal data.length, 2
    assert_equal data[0]['name'], 'Sinclair Target'
    assert_equal data[1]['name'], 'Bob'
  end

  def test_table_csv_email
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'table', '--csv', '-e'
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    assert_equal data.headers, [
      'name', 'email', 'commits', 'last commit time', 'first commit time',
    ]
    assert_equal data.length, 2
    assert_equal data[0]['email'], 'sinclairtarget@gmail.com'
    assert_equal data[1]['email'], 'bob@mail.com'
  end

  def test_table_csv_lines
    cmd = GitWho.new(GitWho.built_bin_path, TestRepo.path)
    stdout_s = cmd.run 'table', '--csv', '-l'
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    assert_equal data.headers, [
      'name',
      'commits',
      'lines added',
      'lines removed',
      'files',
      'last commit time',
      'first commit time',
    ]
    assert_equal data.length, 2
    assert_equal data[0]['name'], 'Sinclair Target'
    assert_equal data[1]['name'], 'Bob'
  end
end
