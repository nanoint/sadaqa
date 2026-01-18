import React from 'react';
import { Home, Heart, User, MessageCircle } from 'lucide-react';

interface LayoutProps {
  children: React.ReactNode;
  activeTab: string;
  onTabChange: (tab: string) => void;
}

export const Layout: React.FC<LayoutProps> = ({ children, activeTab, onTabChange }) => {
  return (
    <div className="min-h-screen bg-gray-50 flex flex-col max-w-md mx-auto shadow-2xl overflow-hidden relative">
      <main className="flex-1 overflow-y-auto pb-24 scrollbar-hide">
        {children}
      </main>

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 w-full max-w-md bg-white border-t border-gray-100 px-6 py-3 flex justify-between items-center z-50 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)]">
        <button
          onClick={() => onTabChange('home')}
          className={`flex flex-col items-center space-y-1 transition-colors ${
            activeTab === 'home' ? 'text-emerald-600' : 'text-gray-400'
          }`}
        >
          <Home size={24} strokeWidth={activeTab === 'home' ? 2.5 : 2} />
          <span className="text-[10px] font-medium">Home</span>
        </button>

        <button
          onClick={() => onTabChange('donate')}
          className={`flex flex-col items-center space-y-1 transition-colors ${
            activeTab === 'donate' ? 'text-emerald-600' : 'text-gray-400'
          }`}
        >
          <Heart size={24} strokeWidth={activeTab === 'donate' ? 2.5 : 2} />
          <span className="text-[10px] font-medium">Donate</span>
        </button>

        <button
          onClick={() => onTabChange('chat')}
          className={`flex flex-col items-center space-y-1 transition-colors ${
            activeTab === 'chat' ? 'text-emerald-600' : 'text-gray-400'
          }`}
        >
          <MessageCircle size={24} strokeWidth={activeTab === 'chat' ? 2.5 : 2} />
          <span className="text-[10px] font-medium">Ask AI</span>
        </button>

        <button
          onClick={() => onTabChange('profile')}
          className={`flex flex-col items-center space-y-1 transition-colors ${
            activeTab === 'profile' ? 'text-emerald-600' : 'text-gray-400'
          }`}
        >
          <User size={24} strokeWidth={activeTab === 'profile' ? 2.5 : 2} />
          <span className="text-[10px] font-medium">Profile</span>
        </button>
      </nav>
    </div>
  );
};