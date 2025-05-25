document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage

    if (!token) {
        // Токена нет
        console.log("No token");
        //window.location.href = '/';
        return;
    }

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:8080/api/getteachercoursesdata', {
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
        fetch('http://localhost:8080/api/getcoursesdata', {
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
            //window.location.href = '/';
        });
    });
});

function handleRedirect() {
    window.location.href = 'http://localhost:8080/profile';
}

function gotonotifications() {
    window.location.href = '/notifications';
}