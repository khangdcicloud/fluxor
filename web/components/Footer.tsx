
import React from 'react';
import { Cpu } from 'lucide-react';

const Footer: React.FC = () => {
  return (
    <footer className="py-16 border-t border-white/5">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center gap-8">
          <div className="flex items-center gap-3">
            <div className="bg-sky-500 p-1.5 rounded-lg">
              <Cpu className="w-6 h-6 text-black" strokeWidth={2.5} />
            </div>
            <span className="text-2xl font-bold tracking-tight">Fluxor</span>
          </div>
          
          <div className="text-zinc-500 text-sm">
            &copy; {new Date().getFullYear()} Fluxor Engine. All rights reserved.
          </div>

          <div className="flex items-center gap-8 text-sm font-medium text-zinc-400">
            <a href="#" className="hover:text-white transition-colors">Documentation</a>
            <a href="#" className="hover:text-white transition-colors">Blog</a>
            <a href="#" className="hover:text-white transition-colors">Community</a>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
