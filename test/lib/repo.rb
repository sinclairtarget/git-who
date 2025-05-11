require 'pathname'

module TestRepo
  def self.path
    p = Pathname.new(__dir__) + '../../test-repo'
    p.cleanpath.to_s
  end
end

# Our bigger test repo with a commit history long enough to require concurrent
# processing.
module BigRepo
  def self.path
    p = Pathname.new(__dir__) + '../../gunicorn'
    p.cleanpath.to_s
  end
end
