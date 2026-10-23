import { describe, expect, it } from 'vitest';
import { isSuspended } from './moderation';

describe('isSuspended', () => {
  it('reads the admin table routes\' text as well as a boolean', () => {
    expect(isSuspended({ username: 'ada', isactive: 'false' })).toBe(true);
    expect(isSuspended({ username: 'ada', isactive: false })).toBe(true);
    expect(isSuspended({ username: 'ada', isactive: 'true' })).toBe(false);
    expect(isSuspended({ username: 'ada', isactive: null })).toBe(false);
  });

  it('never calls the Deleted reader suspended', () => {
    expect(isSuspended({ username: 'deleted.reader', isactive: 'false' })).toBe(false);
  });
});
