import { describe, expect, it } from 'vitest';
import { parseCSV, parseCSVObjects, toCSV } from './csv';

describe('parseCSV', () => {
  it('splits plain rows and fields', () => {
    expect(parseCSV('a,b,c\n1,2,3')).toEqual([['a', 'b', 'c'], ['1', '2', '3']]);
  });

  it('keeps commas, escaped quotes and newlines inside quoted fields', () => {
    const text = 'title,notes\n"Dune, Part 1","He said ""hi""\nthen left"\n';
    expect(parseCSV(text)).toEqual([
      ['title', 'notes'],
      ['Dune, Part 1', 'He said "hi"\nthen left'],
    ]);
  });

  it('handles CRLF line endings and an Excel BOM', () => {
    expect(parseCSV('﻿a,b\r\n1,2\r\n')).toEqual([['a', 'b'], ['1', '2']]);
  });

  it('drops blank lines but keeps empty fields', () => {
    expect(parseCSV('a,b\n\n , \n1,\n')).toEqual([['a', 'b'], ['1', '']]);
  });

  it('keeps a last row with no trailing newline', () => {
    expect(parseCSV('a\nlast')).toEqual([['a'], ['last']]);
  });
});

describe('parseCSVObjects', () => {
  it('keys rows by the trimmed, lower-cased header', () => {
    const text = ' Title ,Author\nDune , Frank Herbert\nEmma';
    expect(parseCSVObjects(text)).toEqual([
      { title: 'Dune', author: 'Frank Herbert' },
      { title: 'Emma', author: '' },
    ]);
  });

  it('returns nothing for an empty file', () => {
    expect(parseCSVObjects('')).toEqual([]);
  });
});

describe('toCSV', () => {
  it('quotes only the fields that need it', () => {
    expect(toCSV([['a', 'b,c', 'say "x"', null, 3]])).toBe('a,"b,c","say ""x""",,3\n');
  });

  it('round-trips through parseCSV', () => {
    const rows = [['title', 'notes'], ['Dune, Part 1', 'line one\nline "two"']];
    expect(parseCSV(toCSV(rows))).toEqual(rows);
  });
});
