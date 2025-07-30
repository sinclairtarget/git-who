require 'csv'
require 'fileutils'
require 'pathname'
require 'tmpdir'

require 'minitest/autorun'

require 'lib/cmd'
require 'lib/repo'

# Tests how git who handles mailmap files.
#
# We read mailmap files from potentially two places: the conventional "local"
# .mailmap file in the repo, but also a "global" mailmap file at an arbitrary
# path pointed to by the mailmap.file git config option.
#
# We want to respect any mappings defined in those files when producing results.
# Because mailmapping might change the author name and email that should be
# attached to any cached commits, changing either the local or global mailmaps
# should invalidate the cache.
class TestMailmap < Minitest::Test
  def teardown
    # Err, cleanup if a test failed and didn't remove this file
    FileUtils.rm_rf Pathname.new(BigRepo.path) / ".mailmap"
  end

  def test_local_mailmap
    Dir.mktmpdir do |dir|
      # Try with no mailmap
      cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
      stdout_s = cmd.run 'table', '--csv', '-e', cache_home: dir
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'

      # Try with mailmap, does it invalidate cache?
      mailmap_path = Pathname.new(BigRepo.path) / ".mailmap"
      File.write(mailmap_path, LOCAL_MAILMAP)

      stdout_s = cmd.run 'table', '--csv', '-e', cache_home: dir
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1322'

      # Try without mailmap again, does it invalidate cache?
      File.delete(mailmap_path)

      stdout_s = cmd.run 'table', '--csv', '-e', cache_home: dir
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'
    end
  end

  def test_global_mailmap
    Dir.mktmpdir do |dir|
      dir = Pathname.new(dir)
      cache_home = dir / ".cache"
      config_home = dir / ".config"
      cache_home.mkdir
      config_home.mkdir

      # Try with no mailmap
      cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'

      # Try with mailmap, does it invalidate cache?
      git_dir = config_home / "git"
      git_dir.mkdir

      mailmap_path = git_dir / ".mailmap"
      File.write(mailmap_path, GLOBAL_MAILMAP)

      git_config_path = git_dir / "config"
      File.write(git_config_path, "[mailmap]\n\tfile = #{mailmap_path}")

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[2]['name'], 'Randall Leeds'
      assert_equal data[2]['email'], 'randall@bleeds.info'
      assert_equal data[2]['commits'], '157'

      # Try without mailmap again, does it invalidate cache?
      File.delete(git_config_path)

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'
    end
  end

  def test_both_mailmap
    Dir.mktmpdir do |dir|
      dir = Pathname.new(dir)
      cache_home = dir / ".cache"
      config_home = dir / ".config"
      cache_home.mkdir
      config_home.mkdir

      # Try with no mailmap
      cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'

      # Try with one mailmap, does it invalidate cache?
      local_mailmap_path = Pathname.new(BigRepo.path) / ".mailmap"
      File.write(local_mailmap_path, LOCAL_MAILMAP)

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1322'
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'

      # Try with two mailmaps, does it invalidate cache?
      git_dir = config_home / "git"
      git_dir.mkdir

      global_mailmap_path = git_dir / ".mailmap"
      File.write(global_mailmap_path, GLOBAL_MAILMAP)

      git_config_path = git_dir / "config"
      File.write(git_config_path, "[mailmap]\n\tfile = #{global_mailmap_path}")

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1322'
      assert_equal data[2]['name'], 'Randall Leeds'
      assert_equal data[2]['email'], 'randall@bleeds.info'
      assert_equal data[2]['commits'], '157'

      # Try with the other mailmap, does it invalidate cache?
      File.delete(local_mailmap_path)

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'
      assert_equal data[2]['name'], 'Randall Leeds'
      assert_equal data[2]['email'], 'randall@bleeds.info'
      assert_equal data[2]['commits'], '157'

      # Try with no mailmap, does it invalidate cache?
      File.delete(git_config_path)

      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'
    end
  end

  # If git config points to a nonexistent file
  def test_bad_configured_global_mailmap_path
    Dir.mktmpdir do |dir|
      dir = Pathname.new(dir)
      cache_home = dir / ".cache"
      config_home = dir / ".config"
      cache_home.mkdir
      config_home.mkdir

      git_dir = config_home / "git"
      git_dir.mkdir
      mailmap_path = git_dir / ".mailmap"
      # NOTE: We aren't creating the file!

      git_config_path = git_dir / "config"
      File.write(git_config_path, "[mailmap]\n\tfile = #{mailmap_path}")

      cmd = GitWho.new(GitWho.built_bin_path, BigRepo.path)
      stdout_s = cmd.run(
        'table',
        '--csv',
        '-e',
        cache_home: cache_home,
        config_home: config_home,
      )
      refute_empty(stdout_s)

      data = CSV.parse(stdout_s, headers: true)
      assert_equal data[0]['name'], 'Benoit Chesneau'
      assert_equal data[0]['email'], 'bchesneau@gmail.com'
      assert_equal data[0]['commits'], '1147'
      assert_equal data[3]['name'], 'Randall Leeds'
      assert_equal data[3]['email'], 'randall@bleeds.info'
      assert_equal data[3]['commits'], '110'
    end
  end
end

LOCAL_MAILMAP = <<~HEREDOC
  Benoit Chesneau <bchesneau@gmail.com> <benoitc@enlil.local>
  Benoit Chesneau <bchesneau@gmail.com> <benoitc@e-engura.org>
  Benoit Chesneau <bchesneau@gmail.com> <benoitc@pollen.nymphormation.org>
HEREDOC

GLOBAL_MAILMAP = <<~HEREDOC
  Randall Leeds <randall@bleeds.info> <randall.leeds@gmail.com>
HEREDOC
