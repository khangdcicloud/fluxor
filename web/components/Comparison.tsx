
import React from 'react';
import { Check, X } from 'lucide-react';

const Comparison: React.FC = () => {
  const rows = [
    { feature: "Concurrency", std: "Manual (go func)", node: "Single Thread", fluxor: "Managed Reactors" },
    { feature: "Safety", std: "Developer's Job", node: "Safe", fluxor: "Safe by Default" },
    { feature: "Performance", std: "High", node: "Low/Medium", fluxor: "Extremely High" },
    { feature: "Architecture", std: "Unopinionated", node: "Callback Hell?", fluxor: "Structured Vert.x" },
    { feature: "Memory Usage", std: "Very Low", node: "High", fluxor: "Native Low" },
  ];

  return (
    <section id="comparison" className="py-24">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <h2 className="text-4xl font-bold mb-4">Why Fluxor?</h2>
          <p className="text-zinc-400">Comparing the leading choices for backend systems.</p>
        </div>

        <div className="overflow-hidden rounded-2xl border border-white/10 bg-white/[0.01] shadow-2xl">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-white/5 border-b border-white/10">
                <th className="px-6 py-4 text-sm font-semibold text-zinc-400">Feature</th>
                <th className="px-6 py-4 text-sm font-semibold">Standard Go</th>
                <th className="px-6 py-4 text-sm font-semibold">Node.js</th>
                <th className="px-6 py-4 text-sm font-semibold text-sky-400">Fluxor (Go)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {rows.map((r, i) => (
                <tr key={i} className="hover:bg-white/[0.02] transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-zinc-300">{r.feature}</td>
                  <td className="px-6 py-4 text-sm text-zinc-500">{r.std}</td>
                  <td className="px-6 py-4 text-sm text-zinc-500">{r.node}</td>
                  <td className="px-6 py-4 text-sm text-sky-400 font-semibold">{r.fluxor}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
};

export default Comparison;
