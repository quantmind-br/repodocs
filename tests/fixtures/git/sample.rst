Sample RST Document
===================

This fixture exercises the constructs supported by ``ConvertRST``.

:author: Test Suite

Introduction
------------

Plain paragraph with **strong**, *emphasis*, and ``inline literal``
plus a :func:`my_function` role and a `Python <https://python.org>`_
hyperlink.

A Subsection
~~~~~~~~~~~~

Bullet list:

* alpha
* beta
* gamma

Enumerated list:

1. first
2. second
#. third

Code Examples
-------------

.. code-block:: python
   :linenos:

   def hello():
       return "world"

Inline literal block::

   raw text
   second line

Admonitions
-----------

.. note::

   Take note of this.

.. warning:: Inline first line.

   Continued body.

Diagrams
--------

.. figure:: assets/flow.png
   :alt: Flow diagram
   :width: 300

.. toctree::
   :maxdepth: 2

   intro
   guide

.. _internal-target: https://example.com/anchor

End of document.
