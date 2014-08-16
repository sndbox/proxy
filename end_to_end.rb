#!/usr/bin/env ruby

# This script requires curl to run

require 'open3'
require 'webrick'

# TODO: run & stop proxy
# TODO: run chunked_responder

def run_server
  logger = WEBrick::Log.new(File::NULL)
  server = WEBrick::HTTPServer.new(
                                   :Port => 18001,
                                   :DocumentRoot => Dir.pwd,
                                   :Logger => logger,
                                   :AccessLog => [])
  Thread.new do |t|
    trap("INT") { server.stop }
    server.start
  end
  return server
end

def do_command(cmd)
  o, e, s = Open3.capture3(cmd)
  raise "#{cmd} failed: #{e}" unless s.success?
  return o
end

def check_diff(a, b)
  class << a; alias :each :each_byte; include Enumerable; end
  class << b; alias :each :each_byte; include Enumerable; end
  a.zip(b).each_with_index do |m, i|
    raise "Diff at #{i}" unless m[0] == m[1]
  end
end

def do_check(arguments)
  begin
    a = do_command("curl -sS -x http://localhost:8082 #{arguments}")
    b = do_command("curl -sS #{arguments}")
    raise "Length mismatch: #{a.length} != #{b.length}" unless a.length == b.length
    check_diff(a, b)
  rescue => e
    STDERR.puts "#{arguments} failed"
    raise e
  end
end

s = run_server()

# Simple GET
do_check("-L http://localhost:18001/")
# Chunked response (assuming test_server running on port 9100)
do_check("http://localhost:9100/chunked?size=22345&delay=600")

s.stop

puts "PASS"
