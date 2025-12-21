
import React from 'react';
import { ArrowRight, Globe, Server, MessageSquare, Database, Cpu, Zap, Shield, Layers, GitBranch } from 'lucide-react';

const Architecture: React.FC = () => {
  return (
    <section id="architecture" className="py-24 bg-[#0d1117]/50 border-y border-white/5">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-20">
          <h2 className="text-4xl font-bold mb-4 tracking-tighter">Under the Hood</h2>
          <p className="text-zinc-400 max-w-2xl mx-auto">
            A reactive framework built on structural concurrency, event-driven architecture, and fail-fast principles.
          </p>
        </div>

        {/* System Architecture Diagram */}
        <div className="mb-20">
          <h3 className="text-2xl font-bold mb-8 text-center">System Architecture</h3>
          <div className="bg-[#0a0a0c] border border-white/10 rounded-2xl p-8">
            <div className="space-y-6">
              {/* Application Layer */}
              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-4">
                  <Layers className="w-6 h-6 text-sky-400" />
                  <h4 className="text-lg font-bold">Application Layer</h4>
                </div>
                <p className="text-sm text-zinc-400 mb-4">User Code: Verticles, Handlers, Workflows, Tasks</p>
                <div className="flex flex-wrap gap-2">
                  <span className="px-3 py-1 bg-sky-500/10 border border-sky-500/20 rounded-full text-xs font-mono text-sky-400">Verticles</span>
                  <span className="px-3 py-1 bg-sky-500/10 border border-sky-500/20 rounded-full text-xs font-mono text-sky-400">Handlers</span>
                  <span className="px-3 py-1 bg-sky-500/10 border border-sky-500/20 rounded-full text-xs font-mono text-sky-400">Workflows</span>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 mx-auto" />

              {/* Fluxor Runtime */}
              <div className="bg-gradient-to-br from-sky-500/10 to-transparent border-2 border-sky-500/30 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-4">
                  <Cpu className="w-6 h-6 text-sky-400" />
                  <h4 className="text-lg font-bold">Fluxor Runtime</h4>
                </div>
                <div className="grid md:grid-cols-3 gap-4">
                  <div className="bg-white/5 border border-white/10 rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <GitBranch className="w-4 h-4 text-blue-400" />
                      <span className="text-sm font-bold">FX</span>
                    </div>
                    <p className="text-xs text-zinc-400">DI/Lifecycle</p>
                  </div>
                  <div className="bg-white/5 border border-white/10 rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <Zap className="w-4 h-4 text-yellow-400" />
                      <span className="text-sm font-bold">Fluxor</span>
                    </div>
                    <p className="text-xs text-zinc-400">Workflows</p>
                  </div>
                  <div className="bg-white/5 border border-white/10 rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <Globe className="w-4 h-4 text-green-400" />
                      <span className="text-sm font-bold">Web</span>
                    </div>
                    <p className="text-xs text-zinc-400">HTTP/WS</p>
                  </div>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 mx-auto" />

              {/* Core Layer */}
              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-4">
                  <Server className="w-6 h-6 text-purple-400" />
                  <h4 className="text-lg font-bold">Core Layer</h4>
                </div>
                <div className="grid md:grid-cols-3 gap-4 mb-4">
                  <div className="bg-zinc-900/50 border border-white/5 rounded-lg p-3 text-center">
                    <span className="text-xs font-mono text-purple-400">Vertx</span>
                  </div>
                  <div className="bg-zinc-900/50 border border-white/5 rounded-lg p-3 text-center">
                    <span className="text-xs font-mono text-blue-400">EventBus</span>
                  </div>
                  <div className="bg-zinc-900/50 border border-white/5 rounded-lg p-3 text-center">
                    <span className="text-xs font-mono text-sky-400">Context</span>
                  </div>
                </div>
                <div className="bg-gradient-to-r from-purple-500/10 to-blue-500/10 border border-purple-500/20 rounded-lg p-4 text-center">
                  <span className="text-sm font-mono text-purple-300">Verticle Deployment</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Core Components */}
        <div className="mb-20">
          <h3 className="text-2xl font-bold mb-8 text-center">Core Components</h3>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-sky-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-sky-500/10 rounded-lg">
                  <Server className="w-5 h-5 text-sky-400" />
                </div>
                <h4 className="font-bold">Vertx</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">Core runtime managing application lifecycle, EventBus, and component registration.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• Start() / Stop()</div>
                <div>• EventBus()</div>
                <div>• RegisterComponent()</div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-blue-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-blue-500/10 rounded-lg">
                  <MessageSquare className="w-5 h-5 text-blue-400" />
                </div>
                <h4 className="font-bold">EventBus</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">Message passing infrastructure for pub/sub, point-to-point, and request-reply messaging.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• Publish() / Send()</div>
                <div>• Request() / Reply()</div>
                <div>• Consumer()</div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-purple-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-purple-500/10 rounded-lg">
                  <Database className="w-5 h-5 text-purple-400" />
                </div>
                <h4 className="font-bold">Worker Pool</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">Bounded worker pools for blocking operations, preventing unbounded goroutine growth.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• Submit()</div>
                <div>• Bounded queues</div>
                <div>• Backpressure</div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-green-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-green-500/10 rounded-lg">
                  <Cpu className="w-5 h-5 text-green-400" />
                </div>
                <h4 className="font-bold">Reactor</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">Single-threaded event loop handling lightweight I/O operations with isolation.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• Non-blocking I/O</div>
                <div>• Actor model</div>
                <div>• Isolation</div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-yellow-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-yellow-500/10 rounded-lg">
                  <Globe className="w-5 h-5 text-yellow-400" />
                </div>
                <h4 className="font-bold">FastHTTPServer</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">High-performance HTTP server with CCU-based backpressure and request ID tracking.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• 100k+ RPS</div>
                <div>• Backpressure</div>
                <div>• Request ID</div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-6 hover:border-red-500/50 transition-all">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-red-500/10 rounded-lg">
                  <Shield className="w-5 h-5 text-red-400" />
                </div>
                <h4 className="font-bold">Fail-Fast</h4>
              </div>
              <p className="text-sm text-zinc-400 mb-3">Immediate error detection and propagation, never silently ignoring errors.</p>
              <div className="space-y-1 text-xs font-mono text-zinc-500">
                <div>• Input validation</div>
                <div>• Error propagation</div>
                <div>• Panic recovery</div>
              </div>
            </div>
          </div>
        </div>

        {/* Request Flow */}
        <div className="mb-20">
          <h3 className="text-2xl font-bold mb-8 text-center">Request Flow</h3>
          <div className="bg-[#0a0a0c] border border-white/10 rounded-2xl p-8">
            <div className="space-y-4">
              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-sky-500/10 border-2 border-sky-500/30 rounded-lg flex items-center justify-center">
                  <span className="text-sky-400 font-bold">1</span>
                </div>
                <div className="flex-1">
                  <h4 className="font-bold mb-1">HTTP Request Arrives</h4>
                  <p className="text-sm text-zinc-400">FastHTTPServer extracts/generates Request ID and creates FastRequestContext</p>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 ml-6" />

              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-blue-500/10 border-2 border-blue-500/30 rounded-lg flex items-center justify-center">
                  <span className="text-blue-400 font-bold">2</span>
                </div>
                <div className="flex-1">
                  <h4 className="font-bold mb-1">Router Matching</h4>
                  <p className="text-sm text-zinc-400">Match route by method + path, extract path parameters</p>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 ml-6" />

              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-purple-500/10 border-2 border-purple-500/30 rounded-lg flex items-center justify-center">
                  <span className="text-purple-400 font-bold">3</span>
                </div>
                <div className="flex-1">
                  <h4 className="font-bold mb-1">Middleware Chain</h4>
                  <p className="text-sm text-zinc-400">Recovery → Observability → Security → Auth → Handler → Response</p>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 ml-6" />

              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-green-500/10 border-2 border-green-500/30 rounded-lg flex items-center justify-center">
                  <span className="text-green-400 font-bold">4</span>
                </div>
                <div className="flex-1">
                  <h4 className="font-bold mb-1">Handler Execution</h4>
                  <p className="text-sm text-zinc-400">Business logic, database queries, EventBus messages, response writing</p>
                </div>
              </div>

              <ArrowRight className="w-6 h-6 text-zinc-600 ml-6" />

              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-yellow-500/10 border-2 border-yellow-500/30 rounded-lg flex items-center justify-center">
                  <span className="text-yellow-400 font-bold">5</span>
                </div>
                <div className="flex-1">
                  <h4 className="font-bold mb-1">HTTP Response</h4>
                  <p className="text-sm text-zinc-400">Update metrics, record span, log response, inject trace context</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Architectural Principles */}
        <div className="p-8 rounded-3xl bg-gradient-to-br from-sky-500/10 to-transparent border border-white/5">
          <h3 className="text-2xl font-bold mb-6 text-center">Architectural Principles</h3>
          <div className="grid md:grid-cols-2 gap-6">
            <div>
              <h4 className="font-bold mb-2 flex items-center gap-2">
                <Shield className="w-5 h-5 text-sky-400" />
                Structural Concurrency
              </h4>
              <p className="text-sm text-zinc-400">Concurrency is designed and enforced with bounded queues, worker pools, and explicit backpressure mechanisms.</p>
            </div>
            <div>
              <h4 className="font-bold mb-2 flex items-center gap-2">
                <Zap className="w-5 h-5 text-yellow-400" />
                Fail-Fast
              </h4>
              <p className="text-sm text-zinc-400">Errors are detected and reported immediately. Input validation happens before processing, errors propagate immediately.</p>
            </div>
            <div>
              <h4 className="font-bold mb-2 flex items-center gap-2">
                <MessageSquare className="w-5 h-5 text-blue-400" />
                Message-First Design
              </h4>
              <p className="text-sm text-zinc-400">Communication happens through message passing via EventBus, not shared state. JSON as default serialization.</p>
            </div>
            <div>
              <h4 className="font-bold mb-2 flex items-center gap-2">
                <Cpu className="w-5 h-5 text-green-400" />
                Framework for Building
              </h4>
              <p className="text-sm text-zinc-400">Provides building blocks, abstractions, and patterns to construct applications without dealing with low-level concurrency.</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default Architecture;
