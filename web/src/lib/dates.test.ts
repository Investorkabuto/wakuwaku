import { describe, expect, it } from 'vitest';
import { localDate } from './dates';

describe('localDate', () => {
	it('zero-pads month and day', () => {
		expect(localDate(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05');
	});
	it('uses local calendar date', () => {
		expect(localDate(new Date(2026, 9, 16, 0, 0))).toBe('2026-10-16');
	});
});
