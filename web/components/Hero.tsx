
import React from 'react';
import { ChevronRight } from 'lucide-react';

const Hero: React.FC = () => {
  return (
    <section className="relative pt-20 pb-16 md:pt-32 md:pb-24 overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col lg:flex-row items-center gap-12">
        
        <div className="flex-1 text-center lg:text-left">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-xs font-semibold mb-6">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-sky-500"></span>
            </span>
            Fluxor v1.0.0-beta released
          </div>
          <h1 className="text-5xl md:text-7xl font-bold tracking-tight mb-6">
            The Reactive Runtime <br />
            <span className="text-sky-500">for Go</span>
          </h1>
          <p className="text-lg md:text-xl text-zinc-400 max-w-2xl mx-auto lg:mx-0 mb-10 leading-relaxed">
            Bring the power of Vert.x and the simplicity of Node.js to the speed of Golang. 
            Deliver 100k+ RPS without a single <code className="text-sky-400 font-mono bg-white/5 px-1.5 rounded">go func()</code> leak.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start">
            <button className="flex items-center justify-center gap-2 bg-sky-500 hover:bg-sky-400 text-black px-8 py-3 rounded-full font-bold text-lg transition-all blue-glow-hover">
              Get Started <ChevronRight className="w-5 h-5" />
            </button>
            <button className="px-8 py-3 rounded-full border border-white/10 hover:bg-white/5 font-bold text-lg transition-all">
              View on GitHub
            </button>
          </div>
        </div>

        <div className="flex-1 w-full max-w-2xl">
          <div className="relative group">
            <div className="absolute -inset-1 bg-gradient-to-r from-sky-500 to-blue-600 rounded-2xl blur opacity-25 group-hover:opacity-40 transition duration-1000 group-hover:duration-200"></div>
            <div className="relative bg-[#0d1117] border border-white/10 rounded-2xl overflow-hidden shadow-2xl">
              <div className="flex items-center gap-2 px-4 py-3 bg-[#161b22] border-b border-white/5">
                <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                <span className="ml-2 text-xs text-zinc-500 font-mono">cmd/main.go</span>
              </div>
              <div className="p-6 overflow-x-auto">
                <pre className="mono text-sm leading-relaxed">
                  <code className="text-sky-400">func</code> <code className="text-white">main() &#123;</code><br />
                  &nbsp;&nbsp;<code className="text-zinc-400">// Initialize runtime</code><br />
                  &nbsp;&nbsp;<code className="text-white">app := fluxor.New()</code><br />
                  &nbsp;&nbsp;<code className="text-white">r := web.NewRouter()</code><br /><br />
                  &nbsp;&nbsp;<code className="text-white">r.GET(</code><code className="text-green-400">"/api/calc"</code><code className="text-white">, func(c *fx.Context) error &#123;</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;<code className="text-zinc-400">// Delegate risk to Worker Pool</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;<code className="text-white">c.Worker().Submit(func() &#123;</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<code className="text-white">res := heavyCalculation()</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<code className="text-white">c.Bus().Publish(</code><code className="text-green-400">"calc.done"</code><code className="text-white">, res)</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;<code className="text-white">&#125;)</code><br />
                  &nbsp;&nbsp;&nbsp;&nbsp;<code className="text-sky-400">return</code> <code className="text-white">c.Ok(fx.JSON&#123;</code><code className="text-green-400">"status"</code><code className="text-white">: </code><code className="text-green-400">"processing"</code><code className="text-white">&#125;)</code><br />
                  &nbsp;&nbsp;<code className="text-white">&#125;)</code><br /><br />
                  &nbsp;&nbsp;<code className="text-white">app.Deploy(web.NewHttpVerticle(</code><code className="text-green-400">"8080"</code><code className="text-white">, r))</code><br />
                  &nbsp;&nbsp;<code className="text-white">app.Run()</code><br />
                  <code className="text-white">&#125;</code>
                </pre>
              </div>
            </div>
          </div>
        </div>
        
      </div>
    </section>
  );
};

export default Hero;
