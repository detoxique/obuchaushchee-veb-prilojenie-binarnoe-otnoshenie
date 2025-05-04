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
        });
    })
    .catch(error => {
        // Ошибка проверки токена
        window.location.href = '/';
    });
});

const form = {
    login: document.getElementById('login'),
    password: document.getElementById('password'),
    role: document.getElementById('role'),
    button: document.querySelector('.submit')
}

form.button.onclick = handleAddUser

async function handleAddUser() {
    const username = form.login.getElementsByTagName('input')[0].value;
    const password = form.password.getElementsByTagName('input')[0].value;
    const role = form.role.getElementsByTagName('select')[0].value;

    try {
        const res = await fetch('/api/admin/adduser', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password, role })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message);

        alert('User added successfully!');
    } catch (err) {
        alert('Error: ' + err.message);
    }
}