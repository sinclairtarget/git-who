require 'csv'
require 'pathname'
require 'tmpdir'

require 'minitest/autorun'

require 'lib/cmd'
require 'lib/repo'


class TestBigRepo < Minitest::Test
  def test_table_csv_big_repo
    cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
    stdout_s = cmd.run 'table', '--csv', n_procs: 1
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    assert_equal data.headers, [
      'name', 'commits', 'last commit time', 'first commit time',
    ]
    assert_equal data.length, 10
    assert_equal data[0]['name'], 'benoitc'
    assert_equal data[0]['commits'], '1043'
  end

  def test_table_csv_big_repo_lines
    cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
    stdout_s = cmd.run 'table', '--csv', '-l'
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    check_sorted_by_lines_csv_results(data)
  end

  def test_table_csv_big_repo_concurrent
    cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
    stdout_s = cmd.run 'table', '--csv', '-l'
    refute_empty(stdout_s)

    data = CSV.parse(stdout_s, headers: true)
    check_sorted_by_lines_csv_results(data)
  end

  def test_table_csv_big_repo_caching
    Dir.mktmpdir do |dir|
      cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)

      git_who_cache_path = Pathname.new(dir) / 'git-who' / 'gob'
      refute git_who_cache_path.exist?

      # First run, cold start
      stdout_s = cmd.run 'table', '--csv', '-l', cache_home: dir
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      check_sorted_by_lines_csv_results(data)

      assert git_who_cache_path.exist?

      # Second run
      stdout_s = cmd.run 'table', '--csv', '-l', cache_home: dir
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      check_sorted_by_lines_csv_results(data)
    end
  end

  def check_sorted_by_lines_csv_results(data)
      assert_equal data.headers, [
        'name',
        'commits',
        'lines added',
        'lines removed',
        'files',
        'last commit time',
        'first commit time',
      ]
      assert_equal data.length, 10

      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['commits'], '316'
      assert_equal data[0]['lines added'], '28094'
      assert_equal data[0]['lines removed'], '24412'
      assert_equal data[0]['files'], '185'

      assert_equal data[1]['name'], 'benoitc'
      assert_equal data[1]['commits'], '1043'
      assert_equal data[1]['lines added'], '28846'
      assert_equal data[1]['lines removed'], '13187'
      assert_equal data[1]['files'], '308'

      assert_equal data[2]['name'], 'Paul J. Davis'
      assert_equal data[2]['commits'], '185'
      assert_equal data[2]['lines added'], '12851'
      assert_equal data[2]['lines removed'], '9117'
      assert_equal data[2]['files'], '264'
  end
end
