require 'pathname'

module Repo
  def self.path
    p = Pathname.new(__dir__) + '../../test-repo'
    p.cleanpath.to_s
  end
end
