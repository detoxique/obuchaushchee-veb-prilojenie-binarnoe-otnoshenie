function gotonotifications() {
    window.location.href = '/notifications';
}

function handleRedirect() {
    window.location.href = 'http://localhost:9293/profile';
}

class Element {
    constructor(name, x, y) {
        this.name = name;
        this.x = x;
        this.y = y;
        this.radius = 20;
    }
}

class Relation {
    constructor(from, to) {
        this.from = from;
        this.to = to;
    }
}

let elements = [];
let relations = [];
let selectedElement = null;
const canvas = document.getElementById('relationCanvas');
const ctx = canvas.getContext('2d');

function drawArrow(from, to) {
    const dx = to.x - from.x;
    const dy = to.y - from.y;
    const angle = Math.atan2(dy, dx);
    const length = Math.sqrt(dx*dx + dy*dy) - from.radius - to.radius;

    const startX = from.x + from.radius * Math.cos(angle);
    const startY = from.y + from.radius * Math.sin(angle);
    const endX = to.x - to.radius * Math.cos(angle);
    const endY = to.y - to.radius * Math.sin(angle);

    // Линия
    ctx.beginPath();
    ctx.moveTo(startX, startY);
    ctx.lineTo(endX, endY);
    ctx.stroke();

    // Стрелка
    const arrowSize = 8;
    ctx.beginPath();
    ctx.moveTo(endX, endY);
    ctx.lineTo(endX - arrowSize*Math.cos(angle - Math.PI/6), endY - arrowSize*Math.sin(angle - Math.PI/6));
    ctx.lineTo(endX - arrowSize*Math.cos(angle + Math.PI/6), endY - arrowSize*Math.sin(angle + Math.PI/6));
    ctx.fill();
}

function drawLoop(element) {
    ctx.beginPath();
    ctx.arc(element.x, element.y + element.radius, 15, 0, 2 * Math.PI);
    ctx.stroke();
    
    // Стрелка петли
    ctx.beginPath();
    ctx.moveTo(element.x + 10, element.y - 5);
    ctx.lineTo(element.x + 15, element.y - 15);
    ctx.lineTo(element.x + 20, element.y - 5);
    ctx.fill();
}

function draw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    // Рисуем отношения
    ctx.strokeStyle = '#333';
    ctx.fillStyle = '#333';
    relations.forEach(rel => {
        if (rel.from === rel.to) {
            drawLoop(rel.from);
        } else {
            drawArrow(rel.from, rel.to);
        }
    });

    // Рисуем элементы
    elements.forEach(element => {
        ctx.beginPath();
        ctx.arc(element.x, element.y, element.radius, 0, 2 * Math.PI);
        ctx.fillStyle = selectedElement === element ? '#ffcccc' : '#fff';
        ctx.fill();
        ctx.stroke();
        
        ctx.fillStyle = '#333';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.font = '16px Arial';
        ctx.fillText(element.name, element.x, element.y);
    });
}

canvas.addEventListener('click', (e) => {
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Проверяем клик по элементу
    const clickedElement = elements.find(el => 
        Math.sqrt((el.x - x)**2 + (el.y - y)**2) < el.radius
    );

    if (clickedElement) {
        if (selectedElement && selectedElement !== clickedElement) {
            relations.push(new Relation(selectedElement, clickedElement));
            selectedElement = null;
        } else {
            selectedElement = clickedElement;
        }
    } else if (selectedElement) {
        relations.push(new Relation(selectedElement, selectedElement)); // Петля
        selectedElement = null;
    }
    
    draw();
});

function addElement() {
    const name = document.getElementById('elementName').value;
    if (name && !elements.some(el => el.name === name)) {
        elements.push(new Element(
            name,
            Math.random() * (canvas.width - 40) + 20,
            Math.random() * (canvas.height - 40) + 20
        ));
        draw();
    }
}

function removeSelected() {
    // Удаление элементов и связанных отношений
    if (selectedElement) {
        elements = elements.filter(el => el !== selectedElement);
        relations = relations.filter(rel => 
            rel.from !== selectedElement && rel.to !== selectedElement
        );
        selectedElement = null;
        draw();
    }
}

// Проверка свойств
function checkReflexivity() {
    const result = elements.every(el => 
        relations.some(rel => rel.from === el && rel.to === el)
    );
    showResult(result ? "Рефлексивное" : "Нерефлексивное", result);
}

function checkSymmetry() {
    const result = relations.every(rel => 
        relations.some(r => r.from === rel.to && r.to === rel.from)
    );
    showResult(result ? "Симметричное" : "Несимметричное", result);
}

function checkTransitivity() {
    const result = relations.every(rel1 => 
        relations.filter(rel2 => rel1.to === rel2.from)
            .every(rel2 => relations.some(rel3 => 
                rel3.from === rel1.from && rel3.to === rel2.to))
    );
    showResult(result ? "Транзитивное" : "Нетранзитивное", result);
}

function showResult(text, isSuccess) {
    const alert = document.getElementById('propertyResult');
    alert.className = `alert alert-${isSuccess ? 'success' : 'danger'}`;
    alert.textContent = text;
}