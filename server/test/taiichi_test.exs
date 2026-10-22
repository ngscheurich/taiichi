defmodule TaiichiTest do
  use ExUnit.Case
  doctest Taiichi

  test "greets the world" do
    assert Taiichi.hello() == :world
  end
end
