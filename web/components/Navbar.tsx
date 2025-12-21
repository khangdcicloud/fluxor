
import React from 'react';
import { Cpu, Github } from 'lucide-react';

const Navbar: React.FC = () => {
  return (
    <nav className="sticky top-0 z-50 w-full border-b border-white/10 bg-[#0a0a0c]/80 backdrop-blur-md">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          <div className="flex items-center gap-2">
            <div className="bg-sky-500 p-1.5 rounded-lg">
              <Cpu className="w-5 h-5 text-black" strokeWidth={2.5} />
            </div>
            <span className="text-xl font-bold tracking-tight">Fluxor</span>
          </div>
          
          <div className="hidden md:flex items-center gap-8 text-sm font-medium text-zinc-400">
            <a href="#features" className="hover:text-white transition-colors">Why Fluxor</a>
            <a href="#architecture" className="hover:text-white transition-colors">Architecture</a>
            <a href="#comparison" className="hover:text-white transition-colors">Comparison</a>
          </div>

          <div className="flex items-center gap-4">
            <a 
              href="https://github.com" 
              className="p-2 text-zinc-400 hover:text-white transition-colors"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github className="w-5 h-5" />
            </a>
            <button className="bg-white text-black text-sm font-semibold py-2 px-4 rounded-full blue-glow hover:bg-sky-50 transition-all">
              Get Started
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
