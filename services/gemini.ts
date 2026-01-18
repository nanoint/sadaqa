import { GoogleGenAI } from "@google/genai";

const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });

export const getSadaqaGuidance = async (query: string): Promise<string> => {
  try {
    const response = await ai.models.generateContent({
      model: 'gemini-3-flash-preview',
      contents: query,
      config: {
        systemInstruction: `You are a helpful and knowledgeable Islamic scholar assistant for a charity app called "Auto Sadaqa". 
        Your goal is to encourage charity (Sadaqa) using wisdom from the Quran and Sunnah, but keep answers concise, warm, and inspiring. 
        Focus on the benefits of consistent small deeds (recurring charity). 
        Format your response in plain text with appropriate emojis. 
        Limit responses to 2-3 short paragraphs.`,
      },
    });

    return response.text || "I apologize, I couldn't generate a response at this moment.";
  } catch (error) {
    console.error("Gemini API Error:", error);
    return "Sorry, I am having trouble connecting to the knowledge base right now.";
  }
};

export const getHadithOfTheDay = async (): Promise<string> => {
    try {
        const response = await ai.models.generateContent({
            model: 'gemini-3-flash-preview',
            contents: "Give me one short, authentic Hadith about charity (Sadaqa) with its reference. Just the text and reference.",
        });
        return response.text || "The upper hand is better than the lower hand (he who gives is better than him who takes).";
    } catch (e) {
        return "The believer's shade on the Day of Resurrection will be his charity. - Al-Tirmidhi";
    }
}