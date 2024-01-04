<script>
    import { endpoint } from '$lib/conf.js'
    import { onMount } from 'svelte';

    let file;
    let status = "Ready to upload :)";
    let link = "";
    let hash;
    let filename;
    let ext;

    let currentDomain;

    onMount(() => currentDomain = window.location.origin);

    async function handleSubmit(event) {
        status = 'Uploading...';
        event.preventDefault();

        filename = file[0].name || 'file.bin';
        ext = filename.split('.').pop() || '.bin';

        const formData = new FormData();
        formData.append('file', file[0]);
        console.log(file[0])

        const response = await fetch(`${endpoint}/store`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            status = `Error: ${response.status} ${response.statusText}`;
            return;
        }

        hash = await response.text();
        status = 'Uploaded successfully! You can download it from the link below:';
        link = `${currentDomain}/f?h=${hash}&e=${ext}&f=${filename}`
    }
</script>

<div class="flex flex-col items-center justify-center min-h-screen bg-gray-100">
    <form on:submit={handleSubmit} class="p-6 mt-10 bg-white rounded shadow-md w-80">
        <div class="flex flex-col">
            <label for="file" class="mb-2 font-bold text-lg text-gray-900">Upload File</label>
            <input id="file" type="file" bind:files={file} required class="p-2 border rounded-md" />
        </div>
        <button type="submit" class="w-full p-2 mt-4 bg-blue-500 hover:bg-blue-700 text-white font-bold rounded">Upload</button>
        <p id="status" class="mt-4 text-center">{status}</p>
        {#if link}
            <a id="link" href={link} class="mt-4 text-center text-blue-500 hover:underline">{link}</a>
        {/if}
    </form>
</div>