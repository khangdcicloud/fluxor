
import React from 'react';
import { ShieldCheck, Zap, BrainCircuit } from 'lucide-react';

const features = [
  {
    icon: <ShieldCheck className="w-8 h-8 text-sky-400" />,
    title: "Safe by Design",
    description: "Forget Race Conditions. Built on the Actor Model (Reactors), Fluxor ensures single-threaded isolation for your logic. No Mutex required."
  },
  {
    icon: <Zap className="w-8 h-8 text-sky-400" />,
    title: "Native Performance",
    description: "100k RPS out of the box. Zero-allocation hot paths. Intelligent Worker Pools. Mimics Vert.x performance with 10x less RAM."
  },
  {
    icon: <BrainCircuit className="w-8 h-8 text-sky-400" />,
    title: "Developer UX First",
    description: "Garbage In, Gold Out. An API designed for Business Logic, not boilerplate. Context-driven development that just works."
  }
];

const Features: React.FC = () => {
  return (
    <section id="features" className="py-24 bg-white/[0.02]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-5xl font-bold mb-4 italic tracking-tight underline decoration-sky-500/30 underline-offset-8">Engineered for Reliability</h2>
          <p className="text-zinc-400 text-lg">Stop fighting the Go memory model. Focus on your business.</p>
        </div>
        
        <div className="grid md:grid-cols-3 gap-8">
          {features.map((f, i) => (
            <div key={i} className="p-8 rounded-2xl border border-white/5 bg-white/[0.01] hover:bg-white/[0.03] hover:border-sky-500/20 transition-all group">
              <div className="mb-6 p-3 bg-sky-500/10 w-fit rounded-xl group-hover:scale-110 transition-transform">
                {f.icon}
              </div>
              <h3 className="text-xl font-bold mb-4">{f.title}</h3>
              <p className="text-zinc-400 leading-relaxed">
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;
