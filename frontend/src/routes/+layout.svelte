<script>
	import { startAnalyticsMonitoring } from "$lib/analytics.js";
	import { onMount } from 'svelte';
	import '../app.css';

	onMount(() => {
		startAnalyticsMonitoring();
	});
</script>

<main
	class="text-text bg-base min-h-screen"
>
	<slot />
</main>

<style>
  main {
    transition: background-color 0.3s, color 0.3s;
  }
</style>