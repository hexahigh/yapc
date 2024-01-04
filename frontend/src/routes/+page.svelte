<script>
    import { endpoint} from '$lib/conf.js'
    import { onMount } from 'svelte';
 
    let file;
    let status;
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
      status = 'Uploaded successfully! You can download it from this link: ' + `${currentDomain}/f?h=${hash}&e=${ext}&f=${filename}`;
    }
 </script>
   
 <form on:submit={handleSubmit}>
    <input type="file" bind:files={file} required />
    <button type="submit">Upload</button>
    <p id="status">{status}</p>
 </form>