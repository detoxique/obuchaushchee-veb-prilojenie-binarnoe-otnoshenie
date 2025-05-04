document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:8080/api/verify', {
        method: 'POST',
        headers: {
            'Authorization': token, // Передаем токен в заголовке
            'Content-Type': 'application/json'
        }
    })
    .then(response => {
        if (!response.ok) {
            window.location.href = '/';
            throw new Error('Token invalid or expired');
        }
        return response.json();
    })
    .then(data => {
        // Успешная проверка токена
        fetch('http://localhost:8080/api/getprofiledata', {
            method: 'POST',
            headers: {
                'Authorization': token, // Передаем токен в заголовке
                'Content-Type': 'application/json'
            }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Token invalid or expired');
            }
            return response.text();
        })
        .then(data => {
            // Страница получена успешно
            console.log(data)
            document.body.innerHTML = data
        })
        .catch(error => {
            // Ошибка проверки токена
            
        });
    })
    .catch(error => {
        // Ошибка проверки токена
        window.location.href = '/';
    });
});