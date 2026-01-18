import React, { useState, useEffect, useRef } from 'react';
import { Layout } from './components/Layout';
import { MosqueCard } from './components/MosqueCard';
import { MOSQUES } from './constants';
import { Mosque, Frequency, DonationPlan, ChatMessage } from './types';
import { initiatePayment, fetchUserPlans } from './services/api';
import { getSadaqaGuidance } from './services/gemini';
import { Search, Loader2, CheckCircle, ArrowLeft, Send, Sparkles, Moon, Heart, Calendar, ShieldCheck } from 'lucide-react';

// Custom Toggle Component
const Toggle = ({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) => (
  <button
    onClick={() => onChange(!checked)}
    className={`w-14 h-8 rounded-full p-1 transition-colors duration-200 ease-in-out ${
      checked ? 'bg-emerald-500' : 'bg-gray-200'
    }`}
  >
    <div
      className={`w-6 h-6 bg-white rounded-full shadow-md transform transition-transform duration-200 ease-in-out ${
        checked ? 'translate-x-6' : 'translate-x-0'
      }`}
    />
  </button>
);

const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState('home');
  const [amount, setAmount] = useState<number>(500);
  const [customAmount, setCustomAmount] = useState<string>('');
  const [isJumaEnabled, setIsJumaEnabled] = useState(true);
  const [paymentStatus, setPaymentStatus] = useState<'idle' | 'processing' | 'success' | 'active'>('idle');
  const [activePlan, setActivePlan] = useState<DonationPlan | null>(null);

  // Profile/Chat State
  const [userPlans, setUserPlans] = useState<DonationPlan[]>([]);
  const [isLoadingPlans, setIsLoadingPlans] = useState(false);
  
  // Chat State
  const [messages, setMessages] = useState<ChatMessage[]>([
    { id: '1', role: 'model', text: 'As-salamu alaykum! How can I help you understand the blessings of Sadaqa today?' }
  ]);
  const [inputMessage, setInputMessage] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Initialize Telegram WebApp
  useEffect(() => {
    if (window.Telegram?.WebApp) {
      window.Telegram.WebApp.ready();
      window.Telegram.WebApp.expand();
      
      // Initial Auth Check
      fetchUserPlans().then(plans => {
        if (plans.length > 0) {
          const weeklyPlan = plans.find(p => p.frequency === Frequency.WEEKLY);
          if (weeklyPlan) {
            setActivePlan(weeklyPlan);
            setPaymentStatus('active');
            setAmount(weeklyPlan.amount);
          }
        }
      });
    }
  }, []);

  // Handle Main Button Logic
  useEffect(() => {
    const tg = window.Telegram?.WebApp;
    if (!tg) return;

    const mainBtn = tg.MainButton;

    if (activeTab === 'home') {
      if (paymentStatus === 'success') {
        mainBtn.hide();
      } else if (paymentStatus === 'active') {
         mainBtn.setText("Manage Subscription");
         mainBtn.setParams({ color: '#ffffff', text_color: '#000000' }); // Secondary style
         mainBtn.show();
      } else {
        // Normal Subscription Flow
        if (isJumaEnabled) {
          mainBtn.setText(`Subscribe ${amount} ₸ / week`);
          mainBtn.setParams({ color: '#10b981', text_color: '#ffffff' });
          mainBtn.show();
          if (paymentStatus === 'processing') {
            mainBtn.showProgress(false);
          } else {
            mainBtn.hideProgress();
          }
        } else {
          mainBtn.hide();
        }
      }
    } else {
      mainBtn.hide();
    }

    const handleMainBtnClick = async () => {
      if (paymentStatus === 'active') {
        // Navigate to profile to manage
        setActiveTab('profile');
        return;
      }

      if (!isJumaEnabled) return;

      setPaymentStatus('processing');
      mainBtn.showProgress(false); // Telegram spins the button loader

      try {
        // Use a default mosque ID for "Auto Sadaqa" general fund or let user select
        // For this specific UI, we assume a default or general pool if mosque not selected,
        // OR we pick the first one. Let's use ID '1' for demo.
        const response = await initiatePayment('1', amount, Frequency.WEEKLY);
        
        // Open Payment Link
        tg.openLink(response.paymentUrl);
        
        // Mock Success after returning
        setTimeout(() => {
          setPaymentStatus('success');
          mainBtn.hideProgress();
          tg.HapticFeedback.notificationOccurred('success');
        }, 1000);
      } catch (e) {
        console.error(e);
        mainBtn.hideProgress();
        setPaymentStatus('idle');
        tg.HapticFeedback.notificationOccurred('error');
      }
    };

    tg.onEvent('mainButtonClicked', handleMainBtnClick);
    return () => {
      tg.offEvent('mainButtonClicked', handleMainBtnClick);
    };
  }, [activeTab, amount, isJumaEnabled, paymentStatus]);

  // Chat Effects
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSendMessage = async () => {
    if (!inputMessage.trim()) return;
    const userMsg: ChatMessage = { id: Date.now().toString(), role: 'user', text: inputMessage };
    setMessages(prev => [...prev, userMsg]);
    setInputMessage('');
    setIsTyping(true);
    const responseText = await getSadaqaGuidance(userMsg.text);
    const modelMsg: ChatMessage = { id: (Date.now() + 1).toString(), role: 'model', text: responseText };
    setMessages(prev => [...prev, modelMsg]);
    setIsTyping(false);
  };

  // Render Functions
  const renderHome = () => {
    if (paymentStatus === 'success') {
      return (
        <div className="flex flex-col items-center justify-center h-[80vh] px-6 text-center animate-in fade-in zoom-in duration-300">
          <div className="w-24 h-24 bg-emerald-100 rounded-full flex items-center justify-center text-emerald-600 mb-6 shadow-lg shadow-emerald-100">
            <CheckCircle size={48} strokeWidth={3} />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Juma Sadaqa Activated!</h2>
          <p className="text-gray-500 mb-8 max-w-xs mx-auto">
            You have successfully subscribed to donate <strong>{amount} ₸</strong> every Friday.
            May Allah accept your deeds.
          </p>
          <button 
            onClick={() => {
              setPaymentStatus('active');
              setActivePlan({
                id: 'new', mosqueId: '1', mosqueName: 'General Sadaqa Fund', 
                amount, currency: 'KZT', frequency: Frequency.WEEKLY, 
                nextPaymentDate: 'Next Friday', status: 'active'
              });
            }}
            className="text-emerald-600 font-semibold hover:bg-emerald-50 px-6 py-3 rounded-xl transition-colors"
          >
            View Subscription
          </button>
        </div>
      );
    }

    if (paymentStatus === 'active' && activePlan) {
      return (
        <div className="pt-8 px-4">
           <div className="bg-emerald-600 rounded-3xl p-6 text-white shadow-xl shadow-emerald-200 relative overflow-hidden mb-6">
              <div className="absolute top-0 right-0 -mt-8 -mr-8 w-32 h-32 bg-white opacity-10 rounded-full blur-2xl"></div>
              <div className="relative z-10">
                <div className="flex items-center space-x-2 mb-4">
                  <ShieldCheck size={20} className="text-emerald-200" />
                  <span className="font-medium text-emerald-100 uppercase tracking-wider text-xs">Active Subscription</span>
                </div>
                <h1 className="text-3xl font-bold mb-1">{activePlan.amount} ₸</h1>
                <p className="text-emerald-100 text-sm">Donated every Friday</p>
                
                <div className="mt-8 pt-4 border-t border-emerald-500/50 flex justify-between items-center">
                  <div className="flex items-center space-x-2">
                    <Calendar size={16} />
                    <span className="text-sm font-medium">Next: {activePlan.nextPaymentDate}</span>
                  </div>
                  <div className="h-2 w-2 bg-emerald-300 rounded-full animate-pulse"></div>
                </div>
              </div>
           </div>
           
           <div className="bg-white rounded-2xl p-6 border border-gray-100 shadow-sm">
             <h3 className="font-semibold text-gray-900 mb-2">Impact</h3>
             <p className="text-gray-500 text-sm leading-relaxed">
               Your consistent contribution helps maintain mosques and supports community programs across Kazakhstan.
             </p>
           </div>
        </div>
      );
    }

    return (
      <div className="pt-6 px-4 pb-24">
        {/* Hero Section */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100 mb-6">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h1 className="text-xl font-bold text-gray-900 flex items-center">
                <Moon size={20} className="mr-2 text-emerald-500" />
                Juma Sadaqa
              </h1>
              <p className="text-gray-500 text-sm mt-1">Automate your Friday good deeds</p>
            </div>
            <Toggle checked={isJumaEnabled} onChange={setIsJumaEnabled} />
          </div>
          
          <div className={`transition-opacity duration-300 ${isJumaEnabled ? 'opacity-100' : 'opacity-50'}`}>
            <p className="text-sm text-gray-600 leading-relaxed bg-gray-50 p-3 rounded-lg border border-gray-100">
              <Sparkles className="inline-block w-4 h-4 text-amber-400 mr-1" />
              "The most beloved of deeds to Allah are those that are most consistent, even if they are small."
            </p>
          </div>
        </div>

        {/* Amount Selector */}
        <div className={`transition-all duration-300 ${isJumaEnabled ? 'opacity-100 translate-y-0' : 'opacity-40 translate-y-2 pointer-events-none'}`}>
          <label className="block text-sm font-semibold text-gray-900 mb-4">
            Choose Amount (Weekly)
          </label>
          
          <div className="grid grid-cols-3 gap-3 mb-4">
            {[200, 500, 1000].map((val) => (
              <button
                key={val}
                onClick={() => {
                  setAmount(val);
                  setCustomAmount('');
                  if(window.Telegram?.WebApp?.HapticFeedback) {
                    window.Telegram.WebApp.HapticFeedback.selectionChanged();
                  }
                }}
                className={`py-4 rounded-xl text-lg font-bold transition-all ${
                  amount === val && !customAmount
                    ? 'bg-emerald-600 text-white shadow-lg shadow-emerald-200 transform scale-[1.02]'
                    : 'bg-white border border-gray-200 text-gray-600 hover:border-emerald-500'
                }`}
              >
                {val}
              </button>
            ))}
          </div>

          <div className="relative">
            <span className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 font-medium">₸</span>
            <input
              type="number"
              value={customAmount}
              onChange={(e) => {
                setCustomAmount(e.target.value);
                setAmount(Number(e.target.value));
              }}
              className={`w-full pl-8 pr-4 py-4 bg-white border rounded-xl focus:ring-2 outline-none transition-all text-lg font-medium ${
                 customAmount ? 'border-emerald-500 ring-emerald-100' : 'border-gray-200'
              }`}
              placeholder="Enter custom amount"
            />
          </div>
          
          <p className="text-center text-xs text-gray-400 mt-6 flex items-center justify-center">
            <ShieldCheck size={12} className="mr-1" />
            Secure payment via Freedom Pay
          </p>
        </div>
      </div>
    );
  };

  const renderProfile = () => (
    <div className="pt-6 px-4">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Profile</h1>
      {/* Mock Profile Content */}
       <div className="bg-gray-900 text-white rounded-2xl p-6 mb-8 shadow-xl">
        <p className="text-gray-400 text-sm mb-1">Total Donations</p>
        <h3 className="text-3xl font-bold">12,500 ₸</h3>
      </div>
      
      <h2 className="text-lg font-bold text-gray-900 mb-4">Subscriptions</h2>
      {activePlan ? (
        <div className="bg-white border border-gray-100 p-4 rounded-xl shadow-sm">
           <p className="font-bold text-gray-900">{activePlan.frequency} Donation</p>
           <p className="text-emerald-600 font-bold text-xl mt-1">{activePlan.amount} ₸</p>
           <p className="text-xs text-gray-400 mt-2">Next payment: {activePlan.nextPaymentDate}</p>
           <button className="mt-4 w-full py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium">Cancel Subscription</button>
        </div>
      ) : (
        <p className="text-gray-500">No active subscriptions.</p>
      )}
    </div>
  );

  const renderChat = () => (
    <div className="flex flex-col h-full bg-white">
      <div className="p-4 border-b border-gray-100 bg-white">
        <h1 className="text-xl font-bold text-gray-900">AI Guide</h1>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-gray-50 pb-20">
        {messages.map(msg => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[80%] p-3 rounded-2xl text-sm ${msg.role === 'user' ? 'bg-emerald-600 text-white' : 'bg-white border shadow-sm'}`}>
              {msg.text}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>
      <div className="p-4 border-t border-gray-100 bg-white fixed bottom-16 left-0 right-0 max-w-md mx-auto">
        <div className="flex items-center space-x-2">
          <input
            value={inputMessage}
            onChange={(e) => setInputMessage(e.target.value)}
            placeholder="Ask about Sadaqa..."
            className="flex-1 bg-gray-100 px-4 py-3 rounded-full focus:outline-none"
          />
          <button onClick={handleSendMessage} className="bg-emerald-600 text-white p-3 rounded-full">
            <Send size={20} />
          </button>
        </div>
      </div>
    </div>
  );

  return (
    <Layout activeTab={activeTab} onTabChange={setActiveTab}>
      {activeTab === 'home' && renderHome()}
      {activeTab === 'donate' && renderHome()} {/* Reuse Home for Donate tab in this simplified flow */}
      {activeTab === 'chat' && renderChat()}
      {activeTab === 'profile' && renderProfile()}
    </Layout>
  );
};

export default App;