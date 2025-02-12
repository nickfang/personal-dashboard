import { readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

type WordEntry = {
  type: string;
  definition: string;
  example: string;
};

type WordDictionary = {
  [key: string]: WordEntry[];
};

const content = readFileSync(join(__dirname, '../src/lib/data/words.txt'), 'utf-8');
const lines = content.split('\n').filter(line => line.trim());

const wordDictionary: WordDictionary = lines.reduce((dict, line) => {
  // Check if line has multiple definitions
  if (line.includes('1.')) {
    const [word, ...rest] = line.split(/\s+(?=\d+\.)/);
    rest.forEach(def => {
      const match = def.match(/(?:\d\.\s+)?\(([^)]+)\)\s+([^(]+)\(([^)]+)\)/);
      if (!match) return;

      const [_, type, definition, example] = match;
      
      if (!dict[word]) {
        dict[word] = [];
      }
      
      dict[word].push({
        type: type.trim(),
        definition: definition.trim(),
        example: example.trim()
      });
    });
    return dict;
  }

  // Handle single definition
  const match = line.match(/^(\w+)\s+\(([^)]+)\)\s+([^(]+)\(([^)]+)\)/);
  if (!match) return dict;

  const [_, word, type, definition, example] = match;
  
  if (!dict[word]) {
    dict[word] = [];
  }
  dict[word].push({
    type: type.trim(),
    definition: definition.trim(),
    example: example.trim()
  });
  
  return dict;
}, {} as WordDictionary);

writeFileSync(
  join(__dirname, '../src/lib/data/sat-words.json'),
  JSON.stringify(wordDictionary, null, 2)
); 