class GitWho
  attr_reader :exit_code

  def initialize(*args)
    @args = args
    @exit_code = 0
  end

  def success?
    @exit_code == 0
  end

  def run
  end
end
