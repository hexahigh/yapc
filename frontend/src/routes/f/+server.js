import { endpoint } from '$lib/conf.js';
import { mime } from 'mime';

export async function GET({ url }) {
   // Extract the HASH and the file extension from the url
   const hash = url.searchParams.get('h') || '0';
   const ext = url.searchParams.get('e') || 'bin';
   const filename = url.searchParams.get('f') || 'file.bin';

   // Construct the URL to the file
   const fileUrl = `${endpoint}/get/${hash}`;

   // Send a GET request to the file URL
   const response = await fetch(fileUrl);

   // Get the content type based on the file extension
   let contentType = mime.getType(ext) || 'application/octet-stream';

   // Return the file with the correct content type and filename
   return {
       headers: {
           'Content-Type': contentType,
           'Content-Disposition': `attachment; filename="${filename}"`
       },
       body: await response.blob()
   };
}