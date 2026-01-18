import React from 'react';
import { MapPin, ChevronRight } from 'lucide-react';
import { Mosque } from '../types';

interface MosqueCardProps {
  mosque: Mosque;
  onSelect: (mosque: Mosque) => void;
}

export const MosqueCard: React.FC<MosqueCardProps> = ({ mosque, onSelect }) => {
  return (
    <div 
      onClick={() => onSelect(mosque)}
      className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden flex flex-row items-center p-3 gap-4 cursor-pointer active:scale-[0.98] transition-transform duration-100"
    >
      <img 
        src={mosque.imageUrl} 
        alt={mosque.name} 
        className="w-20 h-20 object-cover rounded-lg bg-gray-200 shrink-0"
      />
      <div className="flex-1 min-w-0">
        <h3 className="text-base font-semibold text-gray-900 truncate">{mosque.name}</h3>
        <div className="flex items-center text-gray-500 mt-1">
          <MapPin size={14} className="mr-1" />
          <span className="text-xs">{mosque.city}</span>
        </div>
        <p className="text-xs text-gray-400 mt-2 truncate">{mosque.description}</p>
      </div>
      <div className="text-emerald-500">
        <ChevronRight size={20} />
      </div>
    </div>
  );
};