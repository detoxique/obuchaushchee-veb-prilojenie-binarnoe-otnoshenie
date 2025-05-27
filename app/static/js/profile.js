document.addEventListener('DOMContentLoaded', function() {

    const button = document.querySelector('.user-info > button');
    if (button) {
        button.addEventListener('click', function() {
            console.log('handle?');
            localStorage.removeItem('access_token'); // Удаляем токен
            localStorage.removeItem('refresh_token'); // Удаляем токен
            window.location.href = 'http://localhost:9293/';
        });
    }

    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }

    fetch('http://localhost:9293/api/verify', {
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
        return response.json();
    })
    .then(data => {
        // Успешная проверка токена
        console.log(data.message); // Например: "Token valid for user: admin"
        //document.body.innerHTML = `<h2>Logged in</h2>`;
    })
    .catch(error => {
        // Ошибка проверки токена
        //console.error('Error:', error);
        localStorage.removeItem('access_token'); // Удаляем недействительный токен
        //localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
        //setTimeout(() => window.location.href = '/', 5000);

        fetch('http://localhost:9293/api/refreshtoken', {
            method: 'POST',
            headers: {
            'Authorization': localStorage.getItem('refresh_token'), // Передаем токен в заголовке
            'Content-Type': 'application/json'
        }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Token invalid or expired');
            }
            return response.json();
        })
        .then(data => {
            // Успешная проверка токена
            // Сохраняем токен в localStorage
            localStorage.setItem('access_token', data.access_token);
            localStorage.setItem('refresh_token', data.refresh_token);
        })
        .catch(error => {
            // Ошибка проверки токена
            console.error('Error:', error);
            localStorage.removeItem('access_token'); // Удаляем недействительный токен
            localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
        });
    });

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:9293/api/getadminpaneldata', {
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
        fetch('http://localhost:9293/api/getteacherprofiledata', {
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
            fetch('http://localhost:9293/api/getprofiledata', {
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
    window.location.href = 'http://localhost:9293/profile';
}

function logout() {
    localStorage.removeItem('access_token'); // Удаляем токен
    localStorage.removeItem('refresh_token'); // Удаляем токен
    window.location.href = 'http://localhost:9293/';
}

function gotonotifications() {
    window.location.href = '/notifications';
}