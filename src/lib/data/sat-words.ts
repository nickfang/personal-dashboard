export interface SatWord {
  word: string;
  type: string;
  definition: string;
  example: string;
  meaning_number?: number;
}

// This array will be populated by running the convert-words.ts script
export const satWords: SatWord[] = [];

// Function to get word of the day
export function getWordOfDay(): SatWord {
  const today = new Date();
  const dayOfYear = Math.floor(
    (today.getTime() - new Date(today.getFullYear(), 0, 0).getTime()) / 86400000
  );
  return satWords[dayOfYear % satWords.length];
}
