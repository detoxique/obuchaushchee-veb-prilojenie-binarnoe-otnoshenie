const { createApp, ref } = Vue

createApp({
    setup() {
    const username = ref('Username')
    return {
        username
    }
}
}).mount('#app')

const profileButton = document.querySelector('.profile-button');

profileButton.onclick = handleRedirect;

function handleRedirect() {
    window.location.href = 'http://localhost:8080/profile';
}