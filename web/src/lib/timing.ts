import { DoseTiming } from '$lib/gen/wakuwaku/v1/care_pb';

export const TIMINGS = [
	{ value: DoseTiming.MORNING, label: 'あさ', icon: '🌅' },
	{ value: DoseTiming.NOON, label: 'ひる', icon: '☀️' },
	{ value: DoseTiming.EVENING, label: 'ゆう', icon: '🌇' },
	{ value: DoseTiming.BEDTIME, label: 'ねるまえ', icon: '🌙' }
] as const;
