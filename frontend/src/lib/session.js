export function getSessionId() {  
    // Check if a session ID already exists in local storage
    let sessionId = localStorage.getItem('sessionId');

    // If no session ID exists, generate a new one and store it
    if (!sessionId) {
        console.log("No session ID found. Generating a new one.");
        sessionId = generateUniqueId();
        localStorage.setItem('sessionId', sessionId);
    } else {
        console.log("Existing session ID found");
    }

    return sessionId;
}

function generateUniqueId() {
    // Generate a unique ID using a combination of Date.now() and Math.random()
    return 'session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
}