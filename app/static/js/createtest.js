document.addEventListener('DOMContentLoaded', function() {
            const questionsContainer = document.getElementById('questionsContainer');
            const addQuestionBtn = document.getElementById('addQuestionBtn');
            let questionCounter = 0;

            // Добавление нового вопроса
            addQuestionBtn.addEventListener('click', function() {
                questionCounter++;
                const questionId = `question_${questionCounter}`;
                
                const questionBlock = document.createElement('div');
                questionBlock.className = 'question-block';
                questionBlock.innerHTML = `
                    <div class="form-group">
                        <label for="${questionId}_text">Текст вопроса:</label>
                        <textarea id="${questionId}_text" name="questions[${questionCounter}][text]" required></textarea>
                    </div>

                    <div class="form-group">
                        <label for="${questionId}_type">Тип вопроса:</label>
                        <select id="${questionId}_type" name="questions[${questionCounter}][type]" class="question-type" required>
                            <option value="single_choice">Одиночный выбор</option>
                            <option value="multiple_choice">Множественный выбор</option>
                            <option value="text">Текстовый ответ</option>
                            <option value="matching">Сопоставление</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="${questionId}_points">Баллы за вопрос:</label>
                        <input type="number" id="${questionId}_points" name="questions[${questionCounter}][points]" min="1" value="1" required>
                    </div>

                    <!-- Контейнер для вариантов ответов (одиночный/множественный выбор) -->
                    <div class="form-group options-container" id="${questionId}_options">
                        <label>Варианты ответов:</label>
                        <div class="option-list">
                            <div class="option-item">
                                <input type="text" name="questions[${questionCounter}][options][0][text]" placeholder="Текст варианта" required>
                                <input type="checkbox" name="questions[${questionCounter}][options][0][is_correct]">
                                <label>Правильный</label>
                                <button type="button" class="btn btn-small btn-outline-danger remove-option">×</button>
                            </div>
                        </div>
                        <button type="button" class="btn btn-secondary btn-small add-option">Добавить вариант</button>
                    </div>

                    <!-- Контейнер для сопоставления -->
                    <div class="form-group matching-container hidden" id="${questionId}_matching">
                        <label>Пары для сопоставления:</label>
                        <div class="matching-pairs">
                            <div class="matching-item">
                                <input type="text" name="questions[${questionCounter}][left_items][0]" placeholder="Левое значение" required>
                                <span>соответствует</span>
                                <select name="questions[${questionCounter}][right_items][0]" required>
                                    <option value="">Выберите</option>
                                </select>
                                <button type="button" class="btn btn-small btn-outline-danger remove-matching">×</button>
                            </div>
                        </div>
                        <div class="form-group">
                            <label>Варианты для правой колонки:</label>
                            <div class="right-options">
                                <input type="text" class="right-option-input" placeholder="Вариант ответа">
                                <button type="button" class="btn btn-small add-right-option">Добавить</button>
                            </div>
                            <div class="right-options-list"></div>
                        </div>
                        <button type="button" class="btn btn-secondary btn-small add-matching">Добавить пару</button>
                    </div>

                    <div class="form-group">
                        <button type="button" class="btn btn-danger remove-question">Удалить вопрос</button>
                    </div>
                `;

                questionsContainer.appendChild(questionBlock);

                // Элементы вопроса
                const questionType = questionBlock.querySelector('.question-type');
                const optionsContainer = questionBlock.querySelector('.options-container');
                const matchingContainer = questionBlock.querySelector('.matching-container');
                const rightOptionsList = questionBlock.querySelector('.right-options-list');
                let rightOptions = [];

                // Обработчик изменения типа вопроса
                questionType.addEventListener('change', function() {
                    if (this.value === 'text') {
                        optionsContainer.classList.add('hidden');
                        matchingContainer.classList.add('hidden');
                    } else if (this.value === 'matching') {
                        optionsContainer.classList.add('hidden');
                        matchingContainer.classList.remove('hidden');
                    } else {
                        optionsContainer.classList.remove('hidden');
                        matchingContainer.classList.add('hidden');
                        
                        // Обновляем тип input для правильного ответа
                        const correctInputs = questionBlock.querySelectorAll('[name$="[is_correct]"]');
                        correctInputs.forEach(input => {
                            input.type = this.value === 'multiple_choice' ? 'checkbox' : 'radio';
                            input.name = this.value === 'multiple_choice' 
                                ? `questions[${questionCounter}][options][${input.dataset.index}][is_correct]`
                                : `questions[${questionCounter}][correct_option]`;
                        });
                    }
                });

                // Добавление варианта ответа (для одиночного/множественного выбора)
                questionBlock.querySelector('.add-option').addEventListener('click', function() {
                    const optionList = this.previousElementSibling;
                    const optionCount = optionList.children.length;
                    
                    const optionItem = document.createElement('div');
                    optionItem.className = 'option-item';
                    optionItem.innerHTML = `
                        <input type="text" name="questions[${questionCounter}][options][${optionCount}][text]" placeholder="Текст варианта" required>
                        <input type="${questionType.value === 'multiple_choice' ? 'checkbox' : 'radio'}" 
                               name="${questionType.value === 'multiple_choice' 
                                    ? `questions[${questionCounter}][options][${optionCount}][is_correct]` 
                                    : `questions[${questionCounter}][correct_option]`}"
                               value="${optionCount}">
                        <label>${questionType.value === 'multiple_choice' ? 'Правильный' : 'Правильный (только один)'}</label>
                        <button type="button" class="btn btn-small btn-outline-danger remove-option">×</button>
                    `;
                    
                    optionList.appendChild(optionItem);
                });

                // Удаление варианта ответа
                questionBlock.querySelector('.option-list').addEventListener('click', function(e) {
                    if (e.target.classList.contains('remove-option')) {
                        e.target.closest('.option-item').remove();
                    }
                });

                // Добавление варианта для правой колонки (сопоставление)
                questionBlock.querySelector('.add-right-option').addEventListener('click', function() {
                    const input = questionBlock.querySelector('.right-option-input');
                    const value = input.value.trim();
                    
                    if (value) {
                        rightOptions.push(value);
                        updateRightOptions();
                        input.value = '';
                    }
                });

                // Обновление вариантов для правой колонки
                function updateRightOptions() {
                    rightOptionsList.innerHTML = '';
                    rightOptions.forEach((option, index) => {
                        const div = document.createElement('div');
                        div.className = 'option-item';
                        div.innerHTML = `
                            <span>${option}</span>
                            <button type="button" class="btn btn-small btn-outline-danger remove-right-option" data-index="${index}">×</button>
                        `;
                        rightOptionsList.appendChild(div);
                    });

                    // Обновление select во всех парах
                    const selects = questionBlock.querySelectorAll('.matching-pairs select');
                    selects.forEach(select => {
                        const selectedValue = select.value;
                        select.innerHTML = '<option value="">Выберите</option>' + 
                            rightOptions.map((opt, i) => 
                                `<option value="${i}" ${selectedValue === String(i) ? 'selected' : ''}>${opt}</option>`
                            ).join('');
                    });
                }

                // Удаление варианта из правой колонки
                rightOptionsList.addEventListener('click', function(e) {
                    if (e.target.classList.contains('remove-right-option')) {
                        const index = parseInt(e.target.dataset.index);
                        rightOptions.splice(index, 1);
                        updateRightOptions();
                    }
                });

                // Добавление пары сопоставления
                questionBlock.querySelector('.add-matching').addEventListener('click', function() {
                    if (rightOptions.length === 0) {
                        alert('Добавьте варианты для правой колонки сначала!');
                        return;
                    }

                    const pairsContainer = questionBlock.querySelector('.matching-pairs');
                    const pairCount = pairsContainer.children.length;
                    
                    const pairItem = document.createElement('div');
                    pairItem.className = 'matching-item';
                    pairItem.innerHTML = `
                        <input type="text" name="questions[${questionCounter}][left_items][${pairCount}]" placeholder="Левое значение" required>
                        <span>соответствует</span>
                        <select name="questions[${questionCounter}][right_items][${pairCount}]" required>
                            <option value="">Выберите</option>
                            ${rightOptions.map((opt, i) => `<option value="${i}">${opt}</option>`).join('')}
                        </select>
                        <button type="button" class="btn btn-small btn-outline-danger remove-matching">×</button>
                    `;
                    
                    pairsContainer.appendChild(pairItem);
                });

                // Удаление пары сопоставления
                questionBlock.querySelector('.matching-pairs').addEventListener('click', function(e) {
                    if (e.target.classList.contains('remove-matching')) {
                        e.target.closest('.matching-item').remove();
                    }
                });

                // Удаление вопроса
                questionBlock.querySelector('.remove-question').addEventListener('click', function() {
                    questionBlock.remove();
                });
            });

            // Обработка отправки формы
            document.getElementById('testForm').addEventListener('submit', function(e) {
                e.preventDefault();
                
                const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
                // Собираем данные формы
                const formData = {
                    course_id: this.elements.id_course.value,
                    token: token,
                    title: this.elements.name.value,
                    duration: this.elements.duration.value,
                    attempts: this.elements.attempts.value,
                    end_date: this.elements.ends_date.value,
                    questions: []
                };

                // Собираем данные по каждому вопросу
                document.querySelectorAll('.question-block').forEach((questionBlock, qIndex) => {
                    const question = {
                        text: questionBlock.querySelector('textarea').value,
                        type: questionBlock.querySelector('.question-type').value,
                        points: questionBlock.querySelector('input[type="number"]').value
                    };

                    if (question.type === 'matching') {
                        // Обработка вопроса на сопоставление
                        question.left_items = [];
                        question.right_items = [];
                        
                        const leftInputs = questionBlock.querySelectorAll('.matching-item input[type="text"]');
                        const rightSelects = questionBlock.querySelectorAll('.matching-item select');
                        
                        leftInputs.forEach((input, i) => {
                            question.left_items.push(input.value);
                            question.right_items.push(rightSelects[i].value);
                        });
                        
                        question.right_options = [...rightOptions];
                    } else if (question.type !== 'text') {
                        // Обработка вопросов с вариантами ответов
                        question.options = [];
                        
                        questionBlock.querySelectorAll('.option-item').forEach((optionItem, optIndex) => {
                            const option = {
                                text: optionItem.querySelector('input[type="text"]').value,
                                is_correct: question.type === 'multiple_choice' 
                                    ? optionItem.querySelector('input[type="checkbox"]').checked
                                    : optionItem.querySelector('input[type="radio"]').checked
                            };
                            question.options.push(option);
                        });
                    }
                    
                    formData.questions.push(question);
                });

                // Отправка данных на сервер
                fetch('http://localhost:9293/api/tests', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(formData),
                })
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Ошибка сети');
                    }
                    return response.json();
                })
                .then(data => {
                    alert('Тест успешно создан! ID: ' + data.id);
                    // Можно перенаправить на страницу редактирования или список тестов
                })
                .catch(error => {
                    console.error('Ошибка:', error);
                    alert('Произошла ошибка при создании теста');
                });
            });

            // Добавляем первый вопрос при загрузке
            addQuestionBtn.click();
        });