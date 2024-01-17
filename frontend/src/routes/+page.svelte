<script>
	import { endpoint, instanceInfo } from '$lib/conf.js';
	import { onMount } from 'svelte';
	import prettyBytes from 'pretty-bytes';
	import axios from 'axios';
	import autoAnimate from '@formkit/auto-animate';
	import { darkMode } from '$lib/dark.js';

	let files = [];
	let status = 'Ready to upload :)';
	let links = [];
	let hashes = [];
	let filenames = [];
	let exts = [];
	let showInfo = false;
	let uploadCount;
	let ep = endpoint;
	let doArchive = false;

	let totalFiles;
	let totalSize;
	let compression;
	let compressionLevel;
	let server_version;
	let uploadProgress = 0;
	let errorMessage = '';
	let shortenUrl = false;
	let freeSpace;
	let totalSpace;
	let percentageUsed;
	let averageSpeed;

	$: logoSrc = $darkMode ? '/img/logo-dark.svg' : '/img/logo.svg';

	async function getStats() {
		const response = await fetch(`${ep}/stats`);
		const data = await response.json();
		totalFiles = data.totalFiles || 'unknown';
		totalSize = prettyBytes(data.totalSize) || 'unknown';
		compression = data.compression;
		compressionLevel = data.compression_level || 'unknown';
		server_version = data.version || 'unknown';
		freeSpace = prettyBytes(data.availableSpace);
		totalSpace = prettyBytes(data.totalSpace);
		percentageUsed = data.percentageUsed ? parseFloat(data.percentageUsed).toFixed(2) : 'unknown';
		averageSpeed = data.averageSpeed ? prettyBytes(data.averageSpeed) : 'unknown';
	}

	async function archive(url) {
		try {
			const archiveUrl = `https://web.archive.org/save/${url}`;
			const response = await axios.get(archiveUrl);
			if (response.status === 200) {
				const location = response.headers['content-location'];
				const archivedUrl = `https://web.archive.org${location}`;
				console.log(`Page archived at: ${archivedUrl}`);
				return archivedUrl;
			} else {
				console.error(`Failed to archive page. Status: ${response.status}`);
				return null;
			}
		} catch (error) {
			console.error('Error archiving page:', error);
			return null;
		}
	}

	async function shortenLink(url) {
		const payload = {
			url: url
		};

		try {
			const response = await axios.post('https://pomf2.080609.xyz/shorten', payload, {
				headers: {
					//Authorization: 'API-Key f0f2631bbc885aa29Ec204086d9ac32f310Cadd4',
					'Content-Type': 'application/json'
				}
			});

			if (response.data.id) {
				return ep + '/u/' + response.data.id; // Return the shortened URL
			} else {
				console.error('Error shortening URL:', response.data);
				return url; // Return the original URL if there's an error
			}
		} catch (error) {
			console.error('Error shortening URL:', error);
			return url; // Return the original URL if there's an error
		}
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
		uploadProgress = 0;
		errorMessage = '';
		event.preventDefault();

		for (let i = 0; i < files.length; i++) {
			let filename = files[i].name || 'file.bin';
			let ext = filename.split('.').pop() || '.bin';

			const formData = new FormData();
			formData.append('file', files[i]);

			try {
				const response = await axios.post(`${ep}/store`, formData, {
					onUploadProgress: (progressEvent) => {
						const percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total);
						uploadProgress = percentCompleted;
					}
				});

				if (response.status === 200 || response.status === 201) {
					let hash = response.data;
					uploadCount++;
					status = `Uploaded ${uploadCount}/${files.length} files. You can download the latest file from the link below:`;
					let link = encodeURI(`${currentDomain}/f?h=${hash}&e=${ext}&f=${filename}`);
					if (shortenUrl) {
						link = await shortenLink(link);
					}

					if (doArchive) {
						archive(link);
					}
					links = [...links, link];
					filenames = [...filenames, filename];
				} else {
					errorMessage = `Error: ${response.status} ${response.statusText}`;
					break;
				}
			} catch (error) {
				errorMessage = error.message;
				break;
			}
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

<div use:autoAnimate>
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
					class="inline-block align-bottom rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full"
				>
					<div class="bg-base px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
						<div class="sm:flex sm:items-start">
							<div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
								<h3 class="text-lg leading-6 font-medium" id="modal-title">Info</h3>
								<div class="mt-2">
									<p class="text-base text-gray-500">{instanceInfo}</p>
									<p class="text-base text-gray-500">Statistics:</p>
									<p class="text-sm text-gray-500">Server version: {server_version}</p>
									<p class="text-sm text-gray-500">Average server speed: {averageSpeed}</p>
									<p class="text-sm text-gray-500">Total files: {totalFiles}</p>
									<p class="text-sm text-gray-500">Total file size: {totalSize}</p>
									<p class="text-sm text-gray-500">Free space: {freeSpace}</p>
									<p class="text-sm text-gray-500">Total space: {totalSpace}</p>
									<p class="text-sm text-gray-500">Percentage used: {percentageUsed}</p>
									<p class="text-sm text-gray-500">Compression: {compression}</p>
									<p class="text-sm text-gray-500">Compression level: {compressionLevel}</p>
								</div>
							</div>
						</div>
					</div>
					<div class="bg-base px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
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

	<div class="flex flex-col items-center justify-center min-h-screen">
		<img
			src={logoSrc}
			alt="YAPC Logo"
			class="w-64 h-64 pointer-events-none"
			ondragstart="return false;"
		/>
		<form on:submit={handleSubmit} class="p-6 mt-10 rounded shadow-md shadow-white w-80">
			<div class="flex flex-col">
				<label for="file" class="mb-2 font-bold text-lg">Upload Files</label>
				<input id="file" type="file" bind:files multiple required class="p-2 border rounded-md" />			</div>
			<button
				type="submit"
				class="w-full p-2 mt-4 bg-blue-500 hover:bg-blue-700 text-white font-bold rounded"
				>Upload</button
			>
			<label class="flex items-center mt-4">
				<input type="checkbox" bind:checked={shortenUrl} class="form-checkbox" />
				<span class="ml-2"
					>Shorten URL (<a
						href="https://en.wikipedia.org/wiki/Link_rot"
						class="text-blue-500 hover:underline">Not recommended</a
					>)</span
				>
			</label>
			<!--<label class="flex items-center mt-4">
				<input type="checkbox" bind:checked={doArchive} class="form-checkbox" />
				<span class="ml-2"
					>Archive URL</span
				>
			</label>-->
			<p id="status" class="mt-4 text-center">{status}</p>
			{#if uploadProgress > 0 && uploadProgress < 100}
				<progress value={uploadProgress} max="100" class="w-full rounded-md"></progress>
			{/if}
			{#if errorMessage}
				<p class="mt-4 text-center text-red-500">{errorMessage}</p>
			{/if}
		</form>
		<div use:autoAnimate class="mt-10 w-full">
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
		</div>
		<button
			on:click={copyAllToClipboard}
			class="bg-green-500 hover:bg-green-700 text-white font-bold py-1 px-2 rounded"
		>
			Copy All Links
		</button>
	</div>
	<footer class="w-full text-center border-t border-grey p-4 pin-b">
		<a href="https://github.com/hexahigh/yapc" class="hover:underline">Source</a>
		<a href="/terms" class="hover:underline ml-4">Terms</a>
		<button on:click={toggleInfo} class="py-2 px-4 rounded hover:underline"> Info </button>
		<div class="flex justify-center">
			<p class="py-2 px-4">Endpoint:</p>
			<select bind:value={ep} class="py-2 px-4 rounded hover:underline text-white bg-slate-400">
				<option value={endpoint} selected>Main instance</option>
				<option value="http://35.217.17.244:9066">Unlimited (HTTP)</option>
				<option value="http://localhost:8080">Local</option>
			</select>
		</div>
	</footer>
</div>
