#pragma once
#include <SFML/Graphics.hpp>
#include <iostream>

class Button : public sf::Drawable {
private:
	sf::Vector2f pos;
	sf::Vector2f size;
	sf::Color backColor;
	sf::Color foreColor;
	sf::Font font;
	sf::String label;
	int characterSize;
	
	void draw(sf::RenderTarget& target, sf::RenderStates states) const override {
		sf::RectangleShape rect(size);
		rect.setPosition(pos);
		rect.setFillColor(backColor);

		sf::Text text(font, label, characterSize);

		text.setPosition({ pos.x + size.x / 2 - text.getGlobalBounds().size.x / 2, pos.y + size.y / 2 - text.getGlobalBounds().size.y / 2 });
		text.setFillColor(foreColor);

		target.draw(rect, states);
		target.draw(text, states);
	}
public:
	Button() {
		size = sf::Vector2f(80, 30);
		backColor = sf::Color(109, 155, 80, 255);
		foreColor = sf::Color(255, 255, 255, 255);
		characterSize = 14;
	}

	Button(const sf::Vector2f &point) {
		size = sf::Vector2f(80, 30);
		backColor = sf::Color(109, 155, 80, 255);
		foreColor = sf::Color(255, 255, 255, 255);
		pos = point;
		characterSize = 14;
	}

	Button(float x, float y) {
		size = sf::Vector2f(80, 30);
		backColor = sf::Color(109, 155, 80, 255);
		foreColor = sf::Color(255, 255, 255, 255);
		pos = {x, y};
		characterSize = 14;
	}

	Button(const sf::Vector2f& point, const sf::Font& f, const sf::String& text) {
		size = sf::Vector2f(80, 30);
		backColor = sf::Color(109, 155, 80, 255);
		foreColor = sf::Color(255, 255, 255, 255);
		pos = point;
		font = f;
		label = text;
		characterSize = 14;
	}

	void SetPosition(const sf::Vector2f& point) noexcept {
		this->pos = point;
	}

	void SetCharacterSize(int size) noexcept {
		this->characterSize = size;
	}

	sf::RectangleShape GetRect() noexcept {
		sf::RectangleShape rect(size);
		rect.setPosition(pos);
		rect.setFillColor(backColor);

		return rect;
	}
};