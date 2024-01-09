<script>
	import { writable } from 'svelte/store';
	import { darkMode } from '$lib/dark.js';
  import { DollarSolid } from 'flowbite-svelte-icons';
	import '../app.css';

	const currentClass = writable('mocha'); // Default class

	$: darkMode.set($currentClass === 'mocha');

	function toggleClass() {
		currentClass.update((c) => (c === 'mocha' ? 'latte' : 'mocha'));
	}
</script>

<div class="bg-yellow-300 text-center p-2">
  <a href="https://patreon.com/boofdev">Storage does not grow on trees. Consider becoming a Patron.</a>
</div>

<main
	class="text-text bg-base min-h-screen"
	class:mocha={$currentClass === 'mocha'}
	class:latte={$currentClass === 'latte'}
>
<button on:click={toggleClass} class="absolute top-4 right-4 text-2xl w-20 z-10">
	{#if $currentClass === 'mocha'}
		🌑
	{:else}
		☀️
	{/if}
</button>
	<slot />
</main>

<style>
  main {
    transition: background-color 0.3s, color 0.3s;
  }
</style>