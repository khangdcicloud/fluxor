
import React from 'react';
import { ArrowRight, Globe, Server, MessageSquare, Database } from 'lucide-react';

const Architecture: React.FC = () => {
  return (
    <section id="architecture" className="py-24 bg-[#0d1117]/50 border-y border-white/5">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-20">
          <h2 className="text-4xl font-bold mb-4 tracking-tighter">Under the Hood</h2>
          <p className="text-zinc-400 max-w-2xl mx-auto">
            A clean separation of concerns. Reactors handle lightweight I/O, Workers handle the heavy lifting.
          </p>
        </div>

        <div className="flex flex-col md:flex-row items-center justify-between gap-4 md:gap-8 max-w-4xl mx-auto relative">
          
          <div className="flex flex-col items-center gap-2 group w-full md:w-auto">
            <div className="p-6 bg-white/5 border border-white/10 rounded-2xl flex items-center justify-center w-32 h-32 group-hover:border-sky-500/50 transition-all">
              <Globe className="w-10 h-10 text-sky-400" />
            </div>
            <span className="text-sm font-bold">Request</span>
          </div>

          <ArrowRight className="hidden md:block w-8 h-8 text-zinc-700" />

          <div className="flex flex-col items-center gap-2 group w-full md:w-auto">
            <div className="p-6 bg-sky-500/10 border-2 border-sky-500/30 rounded-2xl flex flex-col items-center justify-center w-40 h-40 group-hover:bg-sky-500/20 transition-all shadow-lg shadow-sky-500/10">
              <Server className="w-12 h-12 text-sky-400 mb-2" />
              <span className="text-xs font-mono font-bold text-center">REACTOR<br/>(Single-Threaded)</span>
            </div>
            <span className="text-sm font-bold">Runtime Core</span>
          </div>

          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-3 bg-zinc-900 border border-white/5 p-3 rounded-lg hover:border-blue-500/50 transition-colors">
              <MessageSquare className="w-5 h-5 text-blue-400" />
              <span className="text-xs font-mono">Event Bus</span>
            </div>
            <div className="flex items-center gap-3 bg-zinc-900 border border-white/5 p-3 rounded-lg hover:border-purple-500/50 transition-colors">
              <Database className="w-5 h-5 text-purple-400" />
              <span className="text-xs font-mono">Worker Pool</span>
            </div>
          </div>

        </div>

        <div className="mt-20 p-8 rounded-3xl bg-gradient-to-br from-sky-500/10 to-transparent border border-white/5 text-center">
          <p className="text-lg italic text-zinc-300">
            "Fluxor abstracts the complexity of Go concurrency into a deterministic, high-performance execution model."
          </p>
          <p className="mt-4 font-bold text-sky-500">— Fluxor Architectural Principle</p>
        </div>
      </div>
    </section>
  );
};

export default Architecture;
