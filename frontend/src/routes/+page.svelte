<script>
	import { endpoint } from '$lib/conf.js';
	import { onMount } from 'svelte';
	import prettyBytes from 'pretty-bytes';

	let files = [];
	let status = 'Ready to upload :)';
	let links = [];
	let hashes = [];
	let filenames = [];
	let exts = [];
	let showInfo = false;
	let uploadCount;
	let ep = endpoint;

	let totalFiles;
	let totalSize;
	let compression;
	let compressionLevel;
	let server_version

	async function getStats() {
		const response = await fetch(`${ep}/stats`);
		const data = await response.json();
		totalFiles = data.totalFiles || 'unknown';
		totalSize = prettyBytes(data.totalSize) || 'unknown';
		compression = data.compression;
		compressionLevel = data.compression_level || 'unknown';
		server_version = data.version || 'unknown';
	}

	function toggleInfo() {
		showInfo = !showInfo;
	}

	let currentDomain;

	onMount(async () => {
		currentDomain = window.location.origin;
		await getStats();
	});

	async function handleSubmit(event) {
		uploadCount = 0;
		status = 'Uploading...';
		event.preventDefault();

		for (let i = 0; i < files.length; i++) {
			let filename = files[i].name || 'file.bin';
			let ext = filename.split('.').pop() || '.bin';

			const formData = new FormData();
			formData.append('file', files[i]);
			console.log(files[i]);

			const response = await fetch(`${ep}/store`, {
				method: 'POST',
				body: formData
			});

			if (!response.ok) {
				status = `Error: ${response.status} ${response.statusText}`;
				return;
			}

			let hash = await response.text();
			uploadCount++;
			status = `Uploaded ${uploadCount}/${files.length} files. You can download the latest file from the link below:`;
			let link = encodeURI(`${currentDomain}/f?h=${hash}&e=${ext}&f=${filename}&ep=${ep}`);
			links = [...links, link];
			filenames = [...filenames, filename];
		}
	}

	function copyToClipboard(index) {
		navigator.clipboard.writeText(links[index]);
	}

	function copyAllToClipboard() {
        const allLinks = links.join('\n');
        navigator.clipboard.writeText(allLinks);
    }
</script>

{#if showInfo}
	<div
		class="fixed z-10 inset-0 overflow-y-auto"
		aria-labelledby="modal-title"
		role="dialog"
		aria-modal="true"
	>
		<div
			class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0"
		>
			<div
				class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
				aria-hidden="true"
			></div>
			<span class="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true"
				>&#8203;</span
			>
			<div
				class="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full"
			>
				<div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
					<div class="sm:flex sm:items-start">
						<div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
							<h3 class="text-lg leading-6 font-medium text-gray-900" id="modal-title">Info</h3>
							<div class="mt-2">
								<p class="text-base text-gray-500">Statistics:</p>
								<p class="text-sm text-gray-500">Server version: {server_version}</p>
								<p class="text-sm text-gray-500">Total files: {totalFiles}</p>
								<p class="text-sm text-gray-500">Total file size: {totalSize}</p>
								<p class="text-sm text-gray-500">Compression: {compression}</p>
								<p class="text-sm text-gray-500">Compression level: {compressionLevel}</p>
							</div>
						</div>
					</div>
				</div>
				<div class="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
					<button
						type="button"
						on:click={toggleInfo}
						class="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<div class="flex flex-col items-center justify-center min-h-screen bg-gray-100">
	<img
		class="w-64 h-64 pointer-events-none"
		src="/img/logo.svg"
		ondragstart="return false"
		alt="YAPC logo"
	/>
	<form on:submit={handleSubmit} class="p-6 mt-10 bg-white rounded shadow-md w-80">
		<div class="flex flex-col">
			<label for="file" class="mb-2 font-bold text-lg text-gray-900">Upload Files</label>
			<input id="file" type="file" bind:files multiple required class="p-2 border rounded-md" />
		</div>
		<button
			type="submit"
			class="w-full p-2 mt-4 bg-blue-500 hover:bg-blue-700 text-white font-bold rounded"
			>Upload</button
		>
		<p id="status" class="mt-4 text-center">{status}</p>
	</form>
	<div class="mt-10 w-full">
		{#each links as link, index}
			<div class="ml-11 grid grid-cols-3 gap-4 border-t-2 pt-4 px-4">
				<span class="break-all col-span-1">{filenames[index]}</span>
				<div class="break-all col-span-1">
					<a href={link} class="text-blue-500 hover:underline">{link}</a>
				</div>
				<div class="col-span-1">
					<button
						on:click={() => copyToClipboard(index)}
						type="button"
						class="bg-green-500 hover:bg-green-700 text-white font-bold py-1 px-2 rounded"
						>Copy</button
					>
				</div>
			</div>
		{/each}
		<button on:click={copyAllToClipboard} class="bg-green-500 hover:bg-green-700 text-white font-bold py-1 px-2 rounded"> Copy All Links </button>
	</div>
</div>
<footer class="w-full text-center border-t border-grey p-4 pin-b">
	<a href="https://github.com/hexahigh/yapc" class="hover:underline">Source</a>
	<a href="/terms" class="hover:underline ml-4">Terms</a>
	<button on:click={toggleInfo} class="py-2 px-4 rounded hover:underline"> Info </button>
	<div class="flex justify-center">
		<p class="py-2 px-4">Endpoint:</p>
		<select bind:value={ep} class="py-2 px-4 rounded hover:underline">
			<option value="https://pomf1.080609.xyz" selected>Main instance</option>
			<option value="http://localhost:8080">Local</option>
		</select>

	</div>
</footer>
