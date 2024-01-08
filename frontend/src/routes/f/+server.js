import { endpoint } from '$lib/conf.js';
import mime from 'mime';

export async function GET({ url }) {
   // Extract the HASH and the file extension from the url
   const hash = url.searchParams.get('h') || '0';
   const ext = url.searchParams.get('e') || 'bin';
   const filename = url.searchParams.get('f') || 'file.bin';
   let ep = url.searchParams.get('ep') || endpoint;

   // The main instance is more reliable than the unlimited instance so it should be downloaded from there
    if (ep == "https://tiny-cougar-22.telebit.io") {
        ep = endpoint;
    }

   // Construct the URL to the file
   const fileUrl = `${ep}/get/${hash}`;

   // Send a GET request to the file URL
   const response = await fetch(fileUrl);

   // Get the content type based on the file extension
   let contentType = mime.getType(ext) || 'application/octet-stream';

   // Return the file with the correct content type and filename
   return new Response(response.body, {
    headers: {
        'Content-Type': contentType,
        'Content-Disposition': `attachment; filename="${filename}"`
    },
});
}