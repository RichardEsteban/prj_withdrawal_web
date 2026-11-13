Feature: Depositing cash

  Rule: Customers can only deposit a positive amount

    Scenario: Successful deposit
      Given Alice has 234.56 in their account
      When Alice tries to deposit 100.00
      Then the deposit is successful
      And the new balance is 334.56

    Scenario: Declined deposit of an invalid amount
      Given Hamza has 198.76 in their account
      When Hamza tries to deposit -50.00
      Then the deposit is declined
      And the balance remains 198.76
