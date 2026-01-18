import { PaymentInitResponse, Frequency, DonationPlan } from '../types';

const API_BASE_URL = '/api'; // Relative path for proxy or direct if same origin

// Helper to get Telegram Init Data
const getTelegramInitData = () => {
  if (typeof window !== 'undefined' && window.Telegram?.WebApp) {
    return window.Telegram.WebApp.initData;
  }
  return '';
};

export const initiatePayment = async (
  mosqueId: string,
  amount: number,
  frequency: Frequency
): Promise<PaymentInitResponse> => {
  const initData = getTelegramInitData();
  console.log(`[API] Initiating payment: Mosque=${mosqueId}, Amount=${amount}, Freq=${frequency}`);

  try {
    // In a real scenario, use fetch:
    /*
    const response = await fetch(`${API_BASE_URL}/init-payment`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Telegram-Init-Data': initData,
      },
      body: JSON.stringify({ mosqueId, amount, frequency }),
    });
    if (!response.ok) throw new Error('Payment init failed');
    return await response.json();
    */

    // MOCK IMPLEMENTATION
    await new Promise((resolve) => setTimeout(resolve, 1500));
    
    // Validate initData existence (simulating backend check)
    if (!initData && process.env.NODE_ENV === 'production') {
      console.warn("Missing Telegram Init Data");
      // throw new Error("Unauthorized"); // Commented out for dev/preview without Telegram
    }

    return {
      paymentUrl: 'https://paybox.money/mock-checkout', // Mock URL
      transactionId: `txn_${Math.random().toString(36).substr(2, 9)}`,
    };

  } catch (error) {
    console.error("API Error:", error);
    throw error;
  }
};

export const fetchUserPlans = async (): Promise<DonationPlan[]> => {
  const initData = getTelegramInitData();
  
  try {
    // Mock Delay
    await new Promise((resolve) => setTimeout(resolve, 800));

    // Simulate returning a plan for a specific user ID if needed, 
    // or just return empty for this demo to show the subscription flow.
    // Let's return empty by default to show the "Subscribe" UI.
    return []; 
    
    /* 
    // Example of active plan:
    return [{
      id: 'sub_123',
      mosqueId: '1',
      mosqueName: 'Hazrat Sultan Mosque',
      amount: 1000,
      currency: 'KZT',
      frequency: Frequency.WEEKLY,
      nextPaymentDate: '2025-10-24', // Next Friday
      status: 'active',
    }];
    */
  } catch (error) {
    console.error("Fetch Plans Error:", error);
    return [];
  }
};