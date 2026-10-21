import { describe, expect, it } from 'vitest';
import { formatDate, formatDateTime, formatTime } from '../src/shared/lib/dates';

describe('date formatters', () => {
  it('formats an ISO date for Russian UI', () => {
    expect(formatDate('2026-11-15T00:00:00.000Z')).toMatch(/2026/);
    expect(formatDateTime('2026-11-15T12:00:00.000Z')).toContain('15');
    expect(formatTime('2026-11-15T12:00:00.000Z')).toMatch(/\d{2}:\d{2}/);
  });
});
