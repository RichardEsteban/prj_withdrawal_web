Feature: Transaction integrity and audit log

  Rule: A transaction is all or nothing and every attempt leaves a trace

    Scenario: Successful withdrawal is logged
      Given Alice has 234.56 in their account
      When Alice tries to withdraw 200.00
      Then the transaction is logged as EXITOSA
      And the balance is 34.56

    Scenario: Declined withdrawal is logged
      Given Hamza has 198.76 in their account
      When Hamza tries to withdraw 200.00
      Then the transaction is logged as RECHAZADA
      And the balance remains 198.76

    Scenario: Invalid amount is logged as error
      Given Alice has 234.56 in their account
      When Alice tries to withdraw -20.00
      Then the transaction is logged as ERROR
      And the balance remains 234.56

    Scenario: Transaction is rolled back when the log fails
      Given Alice has 234.56 in their account and the log cannot be written
      When Alice tries to withdraw 200.00
      Then the transaction is rolled back
      And the balance remains 234.56
