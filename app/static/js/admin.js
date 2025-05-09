document.addEventListener('DOMContentLoaded', function() {
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
        // Страница получена успешно
        console.log(data)
        document.body.innerHTML = data
    })
    .catch(error => {
        // Ошибка проверки токена
        document.body.innerHTML = "<h2>Нет доступа<h2>"
        setTimeout(function() {
            window.location.href = '/';
        }, 2000)
    });
});

const form = {
    login: document.getElementsByName('username'),
    password: document.getElementsByName('password'),
    role: document.getElementById('role'),
    group: document.getElementById('groupname')
}

function handleAddUser() {
    const username = form.login[0].value;
    const password = form.password[0].value;
    const role = form.role.value;
    const groupname = form.group.value;
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage

    fetch('http://localhost:8080/api/admin/adduser', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ username, password, role, groupname})
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Token invalid or expired');
        }
        return response.json();
    })
    .then(data => {
        // Страница получена успешно
        console.log(data);
    })
    .catch(error => {
        // Ошибка проверки токена
        alert(error);
    });
}

function logout() {
    localStorage.removeItem('access_token'); // Удаляем токен
            localStorage.removeItem('refresh_token'); // Удаляем токен
            window.location.href = 'http://localhost:8080/';
}