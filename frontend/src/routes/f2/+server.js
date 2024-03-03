import { endpoint } from '$lib/conf.js';

export async function GET({ url, request }) {
	// Extract the HASH and the file extension from the url
	const hash = url.searchParams.get('h') || '0';
	const ext = url.searchParams.get('e') || 'bin';
	const filename = url.searchParams.get('f') || 'file.bin';

    let fileUrl;

	const servers = [
		{
			name: 'NO1',
			url: 'https://pomf1.080609.xyz',
			lat: '59.2083',
			lon: '10.9484'
		}
	];

	const ip =
		request.headers.get('x-forwarded-for') || request.headers.get('remote_addr') || 'unknown';

	if (ip !== 'unknown') {
		// Use ip-api.com to get the latitude and longitude of the IP address
		const response = await fetch(`http://ip-api.com/json/${ip}`);
		const data = await response.json();
		const clientLat = data.lat;
		const clientLon = data.lon;

		// Function to calculate the distance between two points using the Haversine formula
		function calculateDistance(lat1, lon1, lat2, lon2) {
			const R = 6371; // Radius of the earth in km
			const dLat = deg2rad(lat2 - lat1);
			const dLon = deg2rad(lon2 - lon1);
			const a =
				Math.sin(dLat / 2) * Math.sin(dLat / 2) +
				Math.cos(deg2rad(lat1)) * Math.cos(deg2rad(lat2)) * Math.sin(dLon / 2) * Math.sin(dLon / 2);
			const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
			const d = R * c; // Distance in km
			return d;
		}

		function deg2rad(deg) {
			return deg * (Math.PI / 180);
		}

		// Find the closest server
		let closestServer = servers[0];
		let shortestDistance = calculateDistance(
			clientLat,
			clientLon,
			closestServer.lat,
			closestServer.lon
		);

		for (let server of servers) {
			const distance = calculateDistance(clientLat, clientLon, server.lat, server.lon);
			if (distance < shortestDistance) {
				closestServer = server;
				shortestDistance = distance;
			}
		}

		// Function to send a request with a timeout
		async function fetchWithTimeout(url, options, timeout = 2000) {
			return Promise.race([
				fetch(url, options),
				new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), timeout))
			]);
		}

		try {
			// Send a request to the closest server's health endpoint
			await fetchWithTimeout(`${closestServer.url}/health`);
		} catch (error) {
			// If the request fails or times out, switch to another server
			console.log('Failed to reach the closest server, switching to another server...');
			// Implement logic to switch to another server here
			// For simplicity, let's just select the next server in the list
			closestServer = servers.find((server) => server.url !== closestServer.url) || servers[0];
		}

		// Construct the URL to the file on the selected server
		fileUrl = `${closestServer.url}/get2/?h=${hash}&e=${ext}&f=${filename}`;
    } else {
        // Construct the URL to the file on the default server
        fileUrl = `${endpoint}/get2/?h=${hash}&e=${ext}&f=${filename}`;
    }

	// Create a 301 redirect response
	return new Response(null, {
		status: 301,
		headers: {
			Location: fileUrl
		}
	});
}
