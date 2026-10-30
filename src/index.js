// src/index.js
import { Lexer } from './lexer.js';
import { Parser } from './parser.js';
import { Evaluator } from './evaluator.js';
import { TranspilerGo } from './transpiler.js';

export function jalankanKode(sumber) {
  const lexer = new Lexer(sumber);
  const parser = new Parser(lexer);
  const program = parser.parseProgram();

  if (parser.daftarError.length > 0) {
    const errorMsg = parser.daftarError.map(err => `[Galat Sintaks] ${err}`).join('\n');
    throw new Error(errorMsg);
  }

  const evaluator = new Evaluator();
  return evaluator.evaluasi(program);
}

export function transpilasiKodeKeGo(sumber) {
  const lexer = new Lexer(sumber);
  const parser = new Parser(lexer);
  const program = parser.parseProgram();

  if (parser.daftarError.length > 0) {
    const errorMsg = parser.daftarError.map(err => `[Galat Sintaks] ${err}`).join('\n');
    throw new Error(errorMsg);
  }

  const transpiler = new TranspilerGo();
  return transpiler.transpilasi(program);
}

export { Lexer, Parser, Evaluator, TranspilerGo };
