document.addEventListener('DOMContentLoaded', function() {

    const button = document.querySelector('.user-info > button');
    if (button) {
        button.addEventListener('click', function() {
            console.log('handle?');
            localStorage.removeItem('access_token'); // Удаляем токен
            localStorage.removeItem('refresh_token'); // Удаляем токен
            window.location.href = 'http://localhost:8080/';
        });
    }

    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:8080/api/getadminpaneldata', {
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
        // Успешная проверка токена
        window.location.href = '/admin';
        
    })
    .catch(error => {
        // Ошибка проверки токена
        fetch('http://localhost:8080/api/getteacherprofiledata', {
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
            // Успешная проверка токена
            document.body.innerHTML = data;
        })
        .catch(error => {
            // Ошибка проверки токена
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
                // Успешная проверка токена
                document.body.innerHTML = data;
            })
            .catch(error => {
                // Ошибка проверки токена
                console.log(error);
                localStorage.removeItem('access_token'); // Удаляем недействительный токен
                localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
                window.location.href = '/';
            });
        });
    });
});

const profileButton = document.querySelector('.profile-button');

profileButton.onclick = handleRedirect;

function handleRedirect() {
    window.location.href = 'http://localhost:8080/profile';
}

function logout() {
    localStorage.removeItem('access_token'); // Удаляем токен
    localStorage.removeItem('refresh_token'); // Удаляем токен
    window.location.href = 'http://localhost:8080/';
}

function gotonotifications() {
    window.location.href = '/notifications';
}