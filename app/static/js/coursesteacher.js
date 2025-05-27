document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage

    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }
});

function handleRedirect() {
    window.location.href = 'http://localhost:9293/profile';
}