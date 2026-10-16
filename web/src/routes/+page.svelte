<script lang="ts">
	import { timestampDate, timestampNow } from '@bufbuild/protobuf/wkt';
	import { care } from '$lib/api';
	import { localDate } from '$lib/dates';
	import { TIMINGS } from '$lib/timing';
	import type { DoseLog, DoseTiming, Medication } from '$lib/gen/wakuwaku/v1/care_pb';

	type Pending = { med: Medication; timing: DoseTiming; logId: string; previous: DoseLog };

	const today = localDate();
	let meds = $state<Medication[]>([]);
	let logs = $state<DoseLog[]>([]);
	let loading = $state(true);
	let error = $state('');
	let busy = $state(false);
	let pending = $state<Pending | null>(null);

	const slots = $derived(
		TIMINGS.map((t) => ({ ...t, meds: meds.filter((m) => m.timings.includes(t.value)) })).filter(
			(s) => s.meds.length > 0
		)
	);

	function takenLog(medId: string, timing: DoseTiming): DoseLog | undefined {
		return logs.find((l) => l.medicationId === medId && l.timing === timing);
	}

	function clock(log: DoseLog): string {
		if (!log.takenAt) return '';
		return timestampDate(log.takenAt).toLocaleTimeString('ja-JP', {
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	async function load() {
		loading = true;
		error = '';
		try {
			const [m, l] = await Promise.all([
				care.listMedications({}),
				care.listDoseLogs({ fromDate: today, toDate: today })
			]);
			meds = m.medications;
			logs = l.doseLogs;
		} catch {
			error = 'よみこめませんでした。通信をたしかめて、もう一度おためしください。';
		} finally {
			loading = false;
		}
	}

	async function take(med: Medication, timing: DoseTiming, logId: string, confirmDuplicate: boolean) {
		busy = true;
		error = '';
		try {
			const res = await care.recordDose({
				doseLogId: logId,
				medicationId: med.id,
				timing,
				scheduledDate: today,
				takenAt: timestampNow(),
				confirmDuplicate
			});
			if (res.alreadyRecorded && res.doseLog) {
				pending = { med, timing, logId, previous: res.doseLog };
				return;
			}
			pending = null;
			if (res.doseLog) logs = [...logs, res.doseLog];
		} catch {
			error = 'きろくできませんでした。もう一度おためしください。';
		} finally {
			busy = false;
		}
	}

	load();
</script>

<h1>きょうのおくすり</h1>

{#if loading}
	<p>よみこみ中…</p>
{:else if error && meds.length === 0}
	<p class="error" role="alert">{error}</p>
	<button class="btn-secondary" onclick={load}>もう一度よみこむ</button>
{:else if meds.length === 0}
	<p>おくすりが まだ ありません。</p>
{:else}
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#each slots as slot (slot.value)}
		<section class="card slot">
			<h2><span aria-hidden="true">{slot.icon}</span> {slot.label}</h2>
			<ul>
				{#each slot.meds as med (med.id)}
					{@const log = takenLog(med.id, slot.value)}
					<li class:done={log}>
						<div>
							<div class="name">{med.name}</div>
							{#if med.doseText}<div class="dose">{med.doseText}</div>{/if}
						</div>
						{#if log}
							<div class="taken" role="status">✅ のみました（{clock(log)}）</div>
						{:else}
							<button
								class="btn-primary"
								disabled={busy}
								onclick={() => take(med, slot.value, crypto.randomUUID(), false)}
							>
								のんだ
							</button>
						{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/each}
{/if}

<a class="add" href="/medications/new">＋ おくすりを登録する</a>

{#if pending}
	<div class="backdrop">
		<div class="card dialog" role="alertdialog" aria-modal="true" aria-labelledby="dup-title">
			<h2 id="dup-title">さっき きろくしています</h2>
			<p>
				「{pending.med.name}」は {clock(pending.previous)} に「のんだ」と きろくしています。<br />
				ほんとうに もう一度 きろくしますか？
			</p>
			<div class="actions">
				<button class="btn-primary" onclick={() => (pending = null)}>きろくしない</button>
				<button
					class="btn-secondary"
					disabled={busy}
					onclick={() => pending && take(pending.med, pending.timing, pending.logId, true)}
				>
					もう一度きろくする
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	h1 {
		font-size: 1.5rem;
	}
	.slot {
		margin-bottom: 1rem;
	}
	.slot h2 {
		margin: 0 0 0.5rem;
		font-size: 1.3rem;
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.75rem;
	}
	li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.5rem;
		border-radius: var(--radius);
	}
	li.done {
		background: var(--done);
	}
	li > div:first-child {
		flex: 1;
		min-width: 0;
	}
	.name {
		font-weight: 700;
		font-size: 1.15rem;
	}
	.dose {
		color: var(--muted);
	}
	.taken {
		font-weight: 700;
		color: var(--primary);
	}
	li .btn-primary,
	.taken {
		flex-shrink: 0;
	}
	li .btn-primary {
		min-width: 7rem;
		font-size: 1.2rem;
	}
	.add {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: var(--tap);
		margin-top: 1rem;
		border: 2px dashed var(--border);
		border-radius: var(--radius);
		color: var(--primary);
		font-weight: 700;
		text-decoration: none;
	}
	.backdrop {
		position: fixed;
		inset: 0;
		background: rgb(0 0 0 / 0.4);
		display: grid;
		place-items: center;
		padding: 1rem;
	}
	.dialog {
		max-width: 30rem;
	}
	.dialog h2 {
		margin-top: 0;
	}
	.actions {
		display: grid;
		gap: 0.75rem;
	}
</style>
