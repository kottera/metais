# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# The liquid-spec adapter of METAIS's delegated run (ADR-011 D17). It
# selects exactly what liquid-spec.json beside it pins: the suites, the
# strict2 error mode, raised render errors only, the missing features,
# and the departures as known failures, so that liquid-spec itself fails
# a departure that passes. The server is the command given with
# --command. README.md is how to run it; it is written for this
# repository, and nothing of liquid-spec is copied into it.

require "json"
require "liquid/spec/cli/adapter_dsl"
require "liquid/spec/json_rpc/adapter"

PINNED = JSON.parse(File.read(File.join(__dir__, "liquid-spec.json")))

LiquidSpec.setup do |ctx|
  root = Liquid::Spec::CLI::Runner::GEM_ROOT
  head = IO.popen(["git", "-C", root, "rev-parse", "HEAD"], &:read).strip
  unless head == PINNED["commit"]
    raise "liquid-spec at #{root} is at #{head.empty? ? "no commit" : head}, not the pinned #{PINNED["commit"]}"
  end
  command = LiquidSpec.cli_options[:command]
  raise "name the server with --command" if command.nil? || command.empty?

  ctx[:adapter] = Liquid::Spec::JsonRpc::Adapter.new(command, timeout: LiquidSpec.cli_options[:timeout] || 10)
  ctx[:adapter].start
end

LiquidSpec.configure do |config|
  config.suites = PINNED["suites"]
  config.error_modes = [PINNED["error_mode"]]
  config.render_error_modes = [:raise]
  config.missing_features = PINNED["missing_features"]
  config.known_failures = PINNED["departures"].map { |d| d["spec"] }
end

LiquidSpec.compile do |ctx, source, options|
  ctx[:template_id] = ctx[:adapter].compile(source, options)
end

LiquidSpec.render do |ctx, assigns, options|
  ctx[:adapter].render(ctx[:template_id], assigns, options)
end
