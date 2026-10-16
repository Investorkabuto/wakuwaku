<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { session, signIn, signOut } from '$lib/session.svelte';

	let { children } = $props();
	let name = $state('');
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>わくわく農園</title>
</svelte:head>

<header>
	<a href="/" class="brand">🌱 わくわく農園</a>
	{#if session.user}
		<button class="btn-secondary small" onclick={signOut}>ログアウト</button>
	{/if}
</header>

<main>
	{#if session.user}
		{@render children()}
	{:else}
		<form
			class="card login"
			onsubmit={(e) => {
				e.preventDefault();
				if (name.trim()) signIn(name.trim());
			}}
		>
			<h1>ようこそ</h1>
			<p>（開発用ログイン）お名前を入れてください</p>
			<label for="name">お名前</label>
			<input id="name" type="text" bind:value={name} autocomplete="nickname" />
			<button class="btn-primary" type="submit" disabled={!name.trim()}>はじめる</button>
		</form>
	{/if}
</main>

<style>
	header {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		justify-content: space-between;
		padding: 0.5rem 1rem;
		background: var(--primary);
	}
	.brand {
		color: var(--primary-text);
		font-weight: 700;
		font-size: 1.2rem;
		text-decoration: none;
		word-break: keep-all;
		min-height: var(--tap);
		display: flex;
		align-items: center;
	}
	.small {
		font-size: 0.85rem;
		white-space: nowrap;
	}
	main {
		max-width: 40rem;
		margin: 0 auto;
		padding: 1rem;
	}
	.login {
		display: grid;
		gap: 0.75rem;
	}
	.login h1 {
		margin: 0;
	}
	.login p {
		margin: 0;
		color: var(--muted);
	}
</style>
