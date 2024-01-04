<script>
    import { endpoint } from '$lib/conf.js'
    import { onMount } from 'svelte';

    let files = [];
    let status = "Ready to upload :)";
    let links = [];
    let hashes = [];
    let filenames = [];
    let exts = [];

    let currentDomain;

    onMount(() => currentDomain = window.location.origin);

    async function handleSubmit(event) {
        status = 'Uploading...';
        event.preventDefault();

        for (let i = 0; i < files.length; i++) {
            let filename = files[i].name || 'file.bin';
            let ext = filename.split('.').pop() || '.bin';

            const formData = new FormData();
            formData.append('file', files[i]);
            console.log(files[i])

            const response = await fetch(`${endpoint}/store`, {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                status = `Error: ${response.status} ${response.statusText}`;
                return;
            }

            let hash = await response.text();
            status = 'Uploaded successfully! You can download it from the link below:';
            let link = `${currentDomain}/f?h=${hash}&e=${ext}&f=${filename}`
            links = [...links, link];
            filenames = [...filenames, filename];
        }
    }

    function copyToClipboard(index) {
        navigator.clipboard.writeText(links[index]);
    }
</script>

<div class="flex flex-col items-center justify-center min-h-screen bg-gray-100">
    <h1 class="text-3xl font-bold mb-5">YAPC</h1>
    <h3 class="text-2xl font-bold mb-10">Yet another Pomf clone</h3>
    <form on:submit={handleSubmit} class="p-6 mt-10 bg-white rounded shadow-md w-80">
        <div class="flex flex-col">
            <label for="file" class="mb-2 font-bold text-lg text-gray-900">Upload Files</label>
            <input id="file" type="file" bind:files={files} multiple required class="p-2 border rounded-md" />
        </div>
        <button type="submit" class="w-full p-2 mt-4 bg-blue-500 hover:bg-blue-700 text-white font-bold rounded">Upload</button>
        <p id="status" class="mt-4 text-center">{status}</p>
        {#each links as link, index}
            <div class="flex justify-between mt-4">
                <span>{filenames[index]}</span>
                <div>
                    <a href={link} class="text-blue-500 hover:underline">{link}</a>
                    <button on:click={() => copyToClipboard(index)} type="button" class="ml-2 bg-green-500 hover:bg-green-700 text-white font-bold py-1 px-2 rounded">Copy</button>
                </div>
            </div>
        {/each}
    </form>
</div>