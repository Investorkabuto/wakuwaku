<script lang="ts">
	import { goto } from '$app/navigation';
	import { care } from '$lib/api';
	import { TIMINGS } from '$lib/timing';
	import type { DoseTiming } from '$lib/gen/wakuwaku/v1/care_pb';

	let name = $state('');
	let doseText = $state('');
	let timings = $state<DoseTiming[]>([]);
	let saving = $state(false);
	let error = $state('');

	const canSave = $derived(name.trim() !== '' && timings.length > 0 && !saving);

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = '';
		try {
			await care.createMedication({ name, doseText, timings });
			await goto('/');
		} catch {
			error = 'とうろくできませんでした。もう一度おためしください。';
		} finally {
			saving = false;
		}
	}
</script>

<a class="back" href="/">← もどる</a>
<h1>おくすりを登録する</h1>

<form class="card" onsubmit={save}>
	<label for="name">おくすりの名前</label>
	<input id="name" type="text" bind:value={name} maxlength="100" placeholder="れい：血圧のくすり" />

	<label for="dose">1回にのむ量（なくてもOK）</label>
	<input id="dose" type="text" bind:value={doseText} maxlength="100" placeholder="れい：1錠" />

	<fieldset>
		<legend>のむ時間（いくつでも）</legend>
		{#each TIMINGS as t (t.value)}
			<label class="check">
				<input type="checkbox" value={t.value} bind:group={timings} />
				<span aria-hidden="true">{t.icon}</span>
				{t.label}
			</label>
		{/each}
	</fieldset>

	{#if error}<p class="error" role="alert">{error}</p>{/if}
	<button class="btn-primary" type="submit" disabled={!canSave}>
		{saving ? 'とうろく中…' : 'とうろくする'}
	</button>
</form>

<style>
	h1 {
		font-size: 1.5rem;
	}
	.back {
		display: inline-flex;
		align-items: center;
		min-height: var(--tap);
		color: var(--primary);
		font-weight: 700;
		text-decoration: none;
	}
	form {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 0.75rem;
	}
	fieldset {
		border: none;
		padding: 0;
		margin: 0;
		/* fieldset 既定の min-inline-size: min-content によるはみ出しを防ぐ。 */
		min-inline-size: 0;
		display: grid;
		/* 狭い画面では1列にする。 */
		grid-template-columns: repeat(auto-fit, minmax(8.5rem, 1fr));
		gap: 0.5rem;
	}
	legend {
		margin-bottom: 0.5rem;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		min-height: var(--tap);
		padding: 0 0.6rem;
		border: 2px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		cursor: pointer;
		word-break: keep-all;
	}
	.check:has(input:checked) {
		border-color: var(--primary);
		background: var(--done);
		font-weight: 700;
	}
	.check input {
		flex-shrink: 0;
		width: 1.5rem;
		height: 1.5rem;
		accent-color: var(--primary);
	}
</style>
